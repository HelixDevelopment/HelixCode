package shell

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// background_grandchild_grace_test.go — standing regression guard for HXC-198.
//
// THE DEFECT
//
// HXC-184 fixed exactly this hang in ExecuteStream (executor.go). Its sibling
// ExecuteWithProgress — the entry point the background dispatch route uses —
// carried the identical defect and was not fixed:
//
//	wg.Wait()             // background.go:82 — waits for BOTH scanners to EOF
//	waitErr := cmd.Wait() // background.go:84 — only THEN reaps the child
//
// A pipe reaches EOF only once EVERY write end is closed, including the copies
// a GRANDCHILD inherited. For `sleep N & echo started` — or any daemonising
// command — the direct child exits immediately while the grandchild keeps both
// write ends open, so wg.Wait() parks for as long as the grandchild lives.
//
// Nothing rescued it. The read ends came from cmd.StdoutPipe()/StderrPipe(), so
// os/exec owns them and closes them inside cmd.Wait() — which is unreachable,
// sitting behind the very wait that is stuck. And exec.CommandContext's
// cancellation only SIGKILLs the DIRECT child (no Setpgid), which has already
// exited; the grandchild survives holding the descriptors. So there was no time
// limit and no give-up path: not the timeout, not the caller's cancel.
//
// THE CONSEQUENCE
//
// ToolRegistry.adaptToolForBackground (registry.go:1219) routes every
// `run_in_background:true` shell call here via ShellTool.ExecuteWithProgress
// (shell_tools.go:88), so the hang is reachable in practice rather than
// theoretical. A parked call never returns to BackgroundManager.run, so the task
// never reaches a terminal state and its in-flight slot is never released;
// StartTask rejects new work with ErrTooManyTasks once MaxConcurrent (default
// 64) such tasks accumulate.
//
// THE FIX BEING GUARDED
//
// The remedy proven next door in ExecuteStream, applied here: parent-owned
// os.Pipe()s via newStreamPipes (so os/exec never owns or closes the read ends),
// reap the direct child FIRST, then give the scanners a BOUNDED no-progress
// grace (streamDrainGrace, rearmed while lines are still arriving) and on expiry
// close the read ends — which is the only thing that can release a reader parked
// in Read on a pipe a grandchild still holds open. This mirrors
// exec.Cmd.WaitDelay's closeDescriptors(c.parentIOPipes).
//
// §11.4.115 RED-polarity switch: ONE test source, two roles.
//
//	RED_MODE=1 → reproduce the defect on the pre-fix artifact: assert the call
//	             does NOT return within graceDecisionDeadline. The wait is
//	             BOUNDED, so a failing run reports instead of hanging the suite.
//	RED_MODE=0 → standing GREEN guard (default): assert it DOES return, promptly,
//	             with the output produced before the grace expired intact.
//
// §11.4.135: both are ordinary `go test` cases, so they run on every build.

// progressResult carries an ExecuteWithProgress return across the goroutine that
// bounds it, so no test here can ever block the suite indefinitely.
type progressResult struct {
	out interface{}
	err error
}

// runProgressBounded invokes ExecuteWithProgress on its own goroutine and waits
// at most `within` for it to return.
//
// The goroutine is deliberately NOT reaped on timeout: on the pre-fix artifact
// it stays parked in wg.Wait() until the grandchild exits, which is precisely
// the leak being demonstrated. It is bounded by grandchildLifetime, so the test
// binary still exits.
func runProgressBounded(
	ctx context.Context,
	se *ShellExecutor,
	command string,
	sink shellLineSink,
	within time.Duration,
) (*progressResult, time.Duration) {
	done := make(chan progressResult, 1)
	go func() {
		out, err := se.ExecuteWithProgress(ctx, map[string]interface{}{
			"command": command,
		}, sink)
		done <- progressResult{out: out, err: err}
	}()

	start := time.Now()
	select {
	case r := <-done:
		return &r, time.Since(start)
	case <-time.After(within):
		return nil, time.Since(start)
	}
}

// collectingSink returns a sink that accumulates delivered lines, plus a
// snapshot accessor safe to call while the scanners are still running.
func collectingSink() (shellLineSink, func() []string) {
	var (
		mu    sync.Mutex
		lines []string
	)
	sink := func(line string) {
		mu.Lock()
		lines = append(lines, line)
		mu.Unlock()
	}
	snapshot := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), lines...)
	}
	return sink, snapshot
}

// TestExecuteWithProgressReturnsWhenGrandchildHoldsPipes is the HXC-198 guard.
//
// It is the direct analogue of TestExecuteStreamReturnsWhenGrandchildHoldsPipes
// (stream_grandchild_grace_test.go) pointed at the sibling that was missed.
func TestExecuteWithProgressReturnsWhenGrandchildHoldsPipes(t *testing.T) {
	skipIfWindows(t)

	redMode := os.Getenv("RED_MODE") == "1"

	executor := newShellToolInstance()
	sink, delivered := collectingSink()

	// The direct child (sh) exits immediately after echoing. The grandchild
	// (sleep) inherits stdout/stderr and keeps both write ends open, so neither
	// pipe reaches EOF when the child is reaped.
	command := fmt.Sprintf("sleep %d & echo started", int(grandchildLifetime.Seconds()))

	result, elapsed := runProgressBounded(
		context.Background(), executor, command, sink, graceDecisionDeadline)

	if redMode {
		require.Nil(t, result,
			"RED_MODE: expected the BROKEN behaviour (ExecuteWithProgress parks in "+
				"wg.Wait() while the grandchild holds the inherited pipe write ends) "+
				"but it returned after %s — defect NOT reproduced on this artifact", elapsed)
		// The defect is the park itself, so prove delivery had already happened:
		// the scanners are alive and have handed the line over; what is stuck is
		// the wait for their EOF, not the reading.
		assert.Equal(t, []string{"started"}, delivered(),
			"RED_MODE: the line must already have been delivered — this pins the park "+
				"on the un-EOF'd pipe rather than on a scanner that never ran")
		t.Logf("RED_MODE: defect reproduced — ExecuteWithProgress still had not returned %s "+
			"after the direct child exited", elapsed)
		return
	}

	require.NotNil(t, result,
		"ExecuteWithProgress must return within the drain grace even though a "+
			"grandchild still holds the pipe write ends open; it was still blocked after %s",
		elapsed)

	assert.NoError(t, result.err,
		"the direct child exited 0; a deliberate grace-expiry teardown must not be "+
			"reported as a command failure")

	payload, ok := result.out.(map[string]interface{})
	require.True(t, ok, "ExecuteWithProgress returns a map payload")
	assert.Equal(t, 0, payload["exit_code"], "the direct child exited successfully")

	// Output produced BEFORE the grace expired must still be delivered, through
	// the sink and in the aggregated return value alike: the grace must not
	// become "close early and lose output".
	assert.Equal(t, []string{"started"}, delivered(),
		"output written before the grace expired must still reach the sink")
	assert.Equal(t, "started", payload["output"],
		"the aggregated output must carry what the sink received")

	// Bounded truncation is a far better failure mode than an unbounded hang,
	// but it is still a truncation and must be disclosed rather than returned as
	// a partial output that looks complete.
	assert.Equal(t, true, payload["output_incomplete"],
		"the result must honestly disclose that the streams were cut short at the "+
			"grace deadline")
}

// TestExecuteWithProgressCancelReleasesDespiteGrandchild pins the "no give-up
// path" half of HXC-198 specifically.
//
// The existing TestShellBackgroundAware_ContextCancelKillsProcess covers only
// `sleep 30` as the DIRECT child — cancel kills it, its pipes close, and the
// scanners EOF, so that case passed throughout the defect's lifetime. It is the
// GRANDCHILD shape that has no escape: cancellation SIGKILLs the direct child
// which has ALREADY exited, while the grandchild keeps the descriptors, so on
// the pre-fix artifact the caller's own cancel cannot release the call.
//
// That distinction is why this guard exists as well as the one above: it proves
// the fix is a genuine bound and not merely something cancellation would have
// papered over.
func TestExecuteWithProgressCancelReleasesDespiteGrandchild(t *testing.T) {
	skipIfWindows(t)

	redMode := os.Getenv("RED_MODE") == "1"

	executor := newShellToolInstance()
	ctx, cancel := context.WithCancel(context.Background())

	command := fmt.Sprintf("sleep %d & echo started", int(grandchildLifetime.Seconds()))

	// Cancel well before the deadline so the verdict is about cancellation
	// failing to release the call, not about the deadline arriving first.
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()
	defer cancel()

	result, elapsed := runProgressBounded(
		ctx, executor, command, func(string) {}, graceDecisionDeadline)

	if redMode {
		require.Nil(t, result,
			"RED_MODE: expected the BROKEN behaviour (cancelling only SIGKILLs the "+
				"already-exited direct child, so the grandchild keeps the pipes open and "+
				"the call stays parked) but it returned after %s — defect NOT reproduced",
			elapsed)
		t.Logf("RED_MODE: defect reproduced — still parked %s after cancel; "+
			"the caller's own cancel could not release it", elapsed)
		return
	}

	require.NotNil(t, result,
		"a cancelled ExecuteWithProgress must return even when a grandchild holds "+
			"the pipe write ends; it was still blocked after %s", elapsed)
}

// TestExecuteWithProgressDeliversCompleteOutput is the preserve-property guard:
// the bound must not become a truncator.
//
// Asserted unconditionally in BOTH polarities. Full delivery for a normally
// terminating command is a property the fix must PRESERVE, so it must hold on
// the pre-fix and post-fix artifacts alike — a RED_MODE branch here would be
// meaningless. It is the analogue of
// TestExecuteStreamDeliversLargeOutputCompletely, and it is what would catch the
// mistake the first version of the HXC-184 fix made: treating the grace as a
// total budget rather than a no-progress window.
func TestExecuteWithProgressDeliversCompleteOutput(t *testing.T) {
	skipIfWindows(t)

	const lineCount = 5000 // far more than one pipe buffer holds

	executor := newShellToolInstance()
	sink, delivered := collectingSink()

	result, elapsed := runProgressBounded(
		context.Background(), executor,
		fmt.Sprintf("seq 1 %d", lineCount), sink,
		60*time.Second)

	require.NotNil(t, result,
		"a normally-terminating command must complete well inside the deadline (%s)", elapsed)
	require.NoError(t, result.err)

	got := delivered()
	require.Len(t, got, lineCount,
		"every line must be delivered — the grace must not cut a live stream short")
	assert.Equal(t, "1", got[0], "delivery starts at the beginning of the stream")
	assert.Equal(t, fmt.Sprint(lineCount), got[lineCount-1],
		"the stream must run to its true end")

	payload, ok := result.out.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 0, payload["exit_code"], "seq must succeed")
	assert.Equal(t, false, payload["output_incomplete"],
		"a normally-terminating command loses nothing, so the truncation flag must "+
			"stay clear — the disclosure has to be honest in BOTH directions or it "+
			"becomes noise nobody trusts")
}

// TestExecuteWithProgressSlowSinkIsNotTruncated guards the second mistake the
// HXC-184 fix made and had to correct (I1): cutting off a consumer that is
// making progress, just slowly.
//
// Here the sink itself is the slow party — it is invoked synchronously by the
// scanners, so a slow sink stalls reading exactly like a slow channel consumer
// did there. The command exits immediately and there is no grandchild, so the
// ONLY thing that could end this early is a grace treated as a total budget.
// The sink deliberately spends several grace windows in total.
//
// Asserted unconditionally in both polarities, for the same reason as above.
func TestExecuteWithProgressSlowSinkIsNotTruncated(t *testing.T) {
	skipIfWindows(t)

	const (
		lineCount    = 300
		perLineDelay = 20 * time.Millisecond // ~6s total: spans ~3 grace windows
	)

	executor := newShellToolInstance()

	var (
		mu    sync.Mutex
		count int
		last  string
	)
	sink := func(line string) {
		mu.Lock()
		count++
		last = line
		mu.Unlock()
		time.Sleep(perLineDelay)
	}

	result, elapsed := runProgressBounded(
		context.Background(), executor,
		fmt.Sprintf("seq 1 %d", lineCount), sink,
		60*time.Second)

	require.NotNil(t, result, "the command must complete inside the deadline (%s)", elapsed)
	require.NoError(t, result.err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, lineCount, count,
		"a sink that is still consuming must never be cut off for being slow — "+
			"the grace is a no-progress window, not a total drain budget")
	assert.Equal(t, fmt.Sprint(lineCount), last, "the stream must run to its true end")
}

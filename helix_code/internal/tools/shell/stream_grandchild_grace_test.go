package shell

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stream_grandchild_grace_test.go — standing regression guard for HXC-184.
//
// THE DEFECT (regression introduced by fa9f0247)
//
// fa9f0247 correctly replaced Cmd.StdoutPipe()/StderrPipe() with parent-owned
// os.Pipe()s assigned to Cmd.Stdout/Stderr, closing a real race in which
// Cmd.Wait tore the pipes down before the scanners had read a byte. To make
// "Done fired" mean "all output delivered" it then made ExecuteStream block on
// <-streamer.Done() before publishing the result.
//
// <-streamer.Done() requires BOTH scanners to reach EOF, and a pipe reaches EOF
// only once EVERY write end is closed. A grandchild process inherits those
// descriptors. So for `sleep N & echo started` — or any daemonising command —
// the direct child exits immediately but the grandchild keeps the write ends
// open, and ExecuteStream's result goroutine parks on <-streamer.Done() for as
// long as the grandchild lives.
//
// Nothing rescued it: streamer.Stop() was reachable only from the execCtx.Done()
// branch of a select that had already completed, and the timeout callback
// cancelled an execCtx nobody was listening on any more. The deferred cleanup
// never ran, so a semaphore slot (MaxConcurrent, default 10) was consumed and
// never returned — ten occurrences wedge Execute AND ExecuteStream module-wide
// until the process restarts.
//
// This is a genuine regression, not a long-standing gap: the pre-fa9f0247 code
// closed the parent's read ends on reap, so Done fired promptly for the same
// input. The prior fix traded a data-loss race for an unbounded hang.
//
// THE FIX BEING GUARDED
//
// After the direct child is reaped, ExecuteStream waits a BOUNDED grace period
// (streamDrainGrace) for the scanners to finish; on expiry it stops the
// streamer and closes the read ends so any parked Read unblocks, exactly as
// exec.Cmd.WaitDelay does via closeDescriptors(c.parentIOPipes)
// (/usr/lib/golang/src/os/exec/exec.go awaitGoroutines).
//
// §11.4.115 RED-polarity switch: ONE test source, two roles.
//
//	RED_MODE=1 → reproduce the defect on the pre-fix artifact: assert Done does
//	             NOT fire within graceDecisionDeadline. The wait is BOUNDED, so
//	             a failing run reports rather than hanging the suite.
//	RED_MODE=0 → standing GREEN guard (default): assert Done DOES fire within
//	             graceDecisionDeadline and that output produced before the grace
//	             expired is still delivered in full.
//
// §11.4.135: both are ordinary `go test` cases, so they run on every build.

const (
	// grandchildLifetime is how long the orphaned grandchild holds the inherited
	// write ends open. It is chosen to outlive graceDecisionDeadline by a wide
	// margin so neither polarity can be decided by the grandchild exiting.
	grandchildLifetime = 8 * time.Second

	// graceDecisionDeadline is the bounded window both polarities are judged
	// against. It sits between streamDrainGrace (2s — when the FIXED artifact
	// releases) and grandchildLifetime (8s — when even the BROKEN artifact would
	// eventually release), with a 2x margin on each side, so the verdict is not
	// a timing coin-flip on a loaded host.
	graceDecisionDeadline = 4 * time.Second
)

// TestExecuteStreamReturnsWhenGrandchildHoldsPipes is the HXC-184 guard.
func TestExecuteStreamReturnsWhenGrandchildHoldsPipes(t *testing.T) {
	redMode := os.Getenv("RED_MODE") == "1"

	executor := NewShellExecutor(DefaultConfig())

	cmd := &Command{
		ID: "hxc184-grandchild-holds-pipes",
		// The direct child (sh) exits immediately after echoing. The grandchild
		// (sleep) inherits stdout/stderr and keeps both write ends open, so the
		// pipes do not reach EOF when the child is reaped.
		Command: fmt.Sprintf("sleep %d & echo started", int(grandchildLifetime.Seconds())),
	}

	execution, err := executor.ExecuteStream(context.Background(), cmd)
	require.NoError(t, err)

	// Drain concurrently. This is the documented usage contract (see the
	// StreamingExecution doc comment): a consumer that waits on Done before
	// draining can stall the scanners once output exceeds the 100-slot channel
	// buffers. Draining here keeps this guard measuring the grandchild defect
	// and nothing else.
	stdoutLines := make(chan []string, 1)
	go func() {
		var lines []string
		for line := range execution.Stdout {
			lines = append(lines, line)
		}
		stdoutLines <- lines
	}()

	start := time.Now()
	var result *ExecutionResult
	select {
	case result = <-execution.Done:
	case <-time.After(graceDecisionDeadline):
		// Bounded: never an unbounded wait, even when the defect is present.
	}
	elapsed := time.Since(start)

	if redMode {
		require.Nil(t, result,
			"RED_MODE: expected the BROKEN behaviour (Done never fires while the "+
				"grandchild holds the inherited pipe write ends) but Done fired after %s "+
				"— defect NOT reproduced on this artifact", elapsed)
		t.Logf("RED_MODE: defect reproduced — Done still not fired %s after the direct child exited", elapsed)
		return
	}

	require.NotNil(t, result,
		"ExecuteStream must return within the drain grace period even though a "+
			"grandchild still holds the pipe write ends open; it was still blocked after %s",
		elapsed)

	assert.Equal(t, 0, result.ExitCode, "the direct child exited successfully")
	assert.NoError(t, result.Error,
		"a deliberate grace-expiry teardown must not be reported as a stream failure")
	assert.True(t, result.OutputIncomplete,
		"the result must honestly disclose that the streams were cut short at the grace deadline")

	// Output produced BEFORE the grace expired must still be delivered: the
	// grace period must not become "close early and lose output".
	assert.Equal(t, []string{"started"}, <-stdoutLines,
		"output written before the grace expired must still reach the consumer")
}

// TestExecuteStreamHangDoesNotWedgeTheExecutor guards the CONSEQUENCE that makes
// HXC-184 Critical rather than merely slow.
//
// ExecuteStream's deferred cleanup — releasing the semaphore slot, unregistering
// the signal handler, cancelling the timeout, deleting the execution entry —
// sits behind the <-streamer.Done() park. While that park never returns the slot
// is consumed permanently. MaxConcurrent defaults to 10, and the semaphore is
// shared by Execute AND ExecuteStream, so ten such commands stop the executor
// accepting any work at all until the process restarts.
//
// This case compresses that to MaxConcurrent=1 so one occurrence is decisive.
//
//	RED_MODE=1 → assert the follow-up Execute is BLOCKED (slot never returned).
//	RED_MODE=0 → assert the follow-up Execute completes once the grace expires.
func TestExecuteStreamHangDoesNotWedgeTheExecutor(t *testing.T) {
	redMode := os.Getenv("RED_MODE") == "1"

	config := DefaultConfig()
	config.MaxConcurrent = 1 // one wedged execution is enough to exhaust it
	executor := NewShellExecutor(config)

	streamCmd := &Command{
		ID:      "hxc184-slot-wedge-streamer",
		Command: fmt.Sprintf("sleep %d & echo started", int(grandchildLifetime.Seconds())),
	}

	execution, err := executor.ExecuteStream(context.Background(), streamCmd)
	require.NoError(t, err)

	go func() {
		for range execution.Stdout { //nolint:revive // drain so the scanner never parks
		}
	}()

	// Now try to run an unrelated, trivially fast command through the SAME
	// executor. It can only start if the streaming execution gave its slot back.
	ctx, cancel := context.WithTimeout(context.Background(), graceDecisionDeadline)
	defer cancel()

	start := time.Now()
	result, err := executor.Execute(ctx, &Command{ID: "hxc184-slot-wedge-probe", Command: "true"})
	elapsed := time.Since(start)

	if redMode {
		require.Error(t, err,
			"RED_MODE: expected the BROKEN behaviour (the executor is wedged because the "+
				"hung stream never returned its slot) but Execute completed in %s "+
				"— defect NOT reproduced on this artifact", elapsed)
		require.ErrorIs(t, err, context.DeadlineExceeded,
			"RED_MODE: the executor must be wedged on the semaphore, not failing for another reason")
		t.Logf("RED_MODE: defect reproduced — executor still wedged after %s; Execute never acquired a slot", elapsed)
		return
	}

	require.NoError(t, err,
		"the executor must accept new work again once the drain grace expires; "+
			"it was still wedged after %s", elapsed)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.ExitCode, "the probe command must actually run, not merely be admitted")
}

// TestOutputStreamerParksForeverOnBackpressureWithoutStop is the mechanism-level
// proof for the SECOND finding raised alongside HXC-184: a consumer that waits
// on Done BEFORE draining the output channels stalls the scanners once output
// exceeds the 100-slot buffers.
//
// It exercises OutputStreamer directly — no process, no pipes, no timing race —
// so the park is demonstrated deterministically rather than inferred: feed it
// more lines than the buffer holds, never read, and Done provably does not fire;
// then Stop and watch it fire. This is the same unbounded-park mechanism the
// grandchild guard above proves at the executor level, reached by a different
// route (a full channel rather than an un-EOF'd pipe).
//
// Asserted unconditionally in both polarities: the park is inherent to the
// buffered-channel design and is not what HXC-184 changed.
func TestOutputStreamerParksForeverOnBackpressureWithoutStop(t *testing.T) {
	// Comfortably more than the 100-slot channel buffer.
	var payload string
	for i := 0; i < 250; i++ {
		payload += fmt.Sprintf("line-%d\n", i)
	}

	streamer := NewOutputStreamer(strings.NewReader(payload), strings.NewReader(""))
	streamer.Start()

	// Nobody drains GetStdout(). The scanner fills the buffer and parks.
	select {
	case <-streamer.Done():
		t.Fatal("Done fired while the stdout scanner was still parked on a full " +
			"channel — the backpressure contract this documents no longer holds")
	case <-time.After(250 * time.Millisecond):
		// Expected: parked, and would stay parked indefinitely.
	}

	// Stop is the release valve — the one ExecuteStream now reaches on grace
	// expiry instead of parking on <-streamer.Done() forever.
	streamer.Stop()

	select {
	case <-streamer.Done():
	case <-time.After(graceDecisionDeadline):
		t.Fatal("Stop must release a scanner parked on a full output channel")
	}

	assert.ErrorIs(t, streamer.Err(), ErrStreamStopped,
		"an abandoned stream must report ErrStreamStopped, not a genuine I/O failure")
}

// TestExecuteStreamDoneBeforeDrainIsBoundedNotDeadlocked pins what HXC-184's fix
// does to that backpressure hazard end-to-end: a Done-before-drain consumer with
// more output than the buffers hold used to wedge exactly like the grandchild
// case; it now returns within the drain grace with OutputIncomplete set.
//
// Bounded truncation is a strictly better failure mode than an unbounded hang,
// and — unlike the hang — it is disclosed on the result rather than silent. It
// is still a truncation, which is why StreamingExecution documents draining
// concurrently as the supported usage.
func TestExecuteStreamDoneBeforeDrainIsBoundedNotDeadlocked(t *testing.T) {
	const lineCount = 500 // > the 100-slot stdout buffer

	executor := NewShellExecutor(DefaultConfig())

	execution, err := executor.ExecuteStream(context.Background(), &Command{
		ID:      "hxc184-done-before-drain",
		Command: fmt.Sprintf("seq 1 %d", lineCount),
	})
	require.NoError(t, err)

	// Deliberately the DISCOURAGED usage: wait on Done without draining first.
	var result *ExecutionResult
	select {
	case result = <-execution.Done:
	case <-time.After(graceDecisionDeadline):
	}

	require.NotNil(t, result,
		"a Done-before-drain consumer must not deadlock; it must be released by the drain grace")
	assert.True(t, result.OutputIncomplete,
		"truncation caused by an undrained consumer must be disclosed, never silent")
	assert.NoError(t, result.Error,
		"a deliberate teardown is ErrStreamStopped-class and must not surface as a stream failure")

	var lines []string
	for line := range execution.Stdout {
		lines = append(lines, line)
	}
	require.NotEmpty(t, lines, "the lines buffered before the scanner parked must still be delivered")
	assert.Equal(t, "1", lines[0], "delivery starts at the beginning of the stream")
	assert.Less(t, len(lines), lineCount,
		"this case is only meaningful if the output genuinely exceeded the buffer")
}

// TestExecuteStreamDeliversLargeOutputCompletely proves the HXC-184 fix does not
// reintroduce the data-loss race fa9f0247 closed, and does not truncate a large
// stream at the new grace deadline.
//
// It asserts unconditionally in BOTH polarities: full delivery is the property
// the fix must PRESERVE, so it must hold on the pre-fix and post-fix artifacts
// alike. A RED_MODE branch here would be meaningless.
func TestExecuteStreamDeliversLargeOutputCompletely(t *testing.T) {
	const lineCount = 100000

	executor := NewShellExecutor(DefaultConfig())

	cmd := &Command{
		ID:      "hxc184-large-output-completeness",
		Command: fmt.Sprintf("seq 1 %d", lineCount),
	}

	execution, err := executor.ExecuteStream(context.Background(), cmd)
	require.NoError(t, err)

	// Drain concurrently with the child: ~589 KB cannot fit the 64 KB pipe
	// buffer, so the child only makes progress while someone is reading.
	type drained struct {
		count int
		first string
		last  string
	}
	got := make(chan drained, 1)
	go func() {
		var d drained
		for line := range execution.Stdout {
			if d.count == 0 {
				d.first = line
			}
			d.last = line
			d.count++
		}
		got <- d
	}()

	var result *ExecutionResult
	select {
	case result = <-execution.Done:
	case <-time.After(60 * time.Second):
		t.Fatal("ExecuteStream did not complete within 60s for a large-output command")
	}

	require.NoError(t, result.Error)
	require.Equal(t, 0, result.ExitCode, "seq must succeed")
	assert.False(t, result.OutputIncomplete,
		"a normally-terminating command must not be reported as truncated")

	d := <-got
	assert.Equal(t, "1", d.first, "the first line must survive the parent-owned-pipe wiring")
	assert.Equal(t, fmt.Sprint(lineCount), d.last,
		"the last line must survive: the grace period must not cut a live stream short")
	assert.Equal(t, lineCount, d.count,
		"every line must be delivered — no gap in the middle")
}

package shell

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// executor_bounded_wait_test.go — standing regression guard for HXC-211.
//
// HXC-184 (ExecuteStream) and HXC-198 (ExecuteWithProgress) each fixed one
// instance of "this call waits for something that may never happen". A sweep of
// the package found the same shape surviving in DefaultExecutor.Execute, in two
// independent forms. Both are guarded here.
//
// The sweep also found the shape in OutputStreamer, where it is NOT a defect but
// a contract the one production caller happens to discharge. That contract is
// pinned at the end of this file, so the next caller cannot quietly break it.
//
// ─── DEFECT A: the SIGKILL never reaches the process ────────────────────────
//
// Execute's cancel/timeout branch is:
//
//	case <-execCtx.Done():
//	    e.signalHandler.Send(cmd.ID, syscall.SIGKILL)   // error DISCARDED
//	    ...
//	    <-done                                          // wait for it to exit
//
// SignalHandler.Send signals a process GROUP whenever the registered PGID is
// positive (sandbox.go, `if info.PGID > 0 { pid = -info.PGID }`), and Execute
// registered a PGID that was ALWAYS positive:
//
//	pgid := pid
//	if execCmd.SysProcAttr != nil && execCmd.SysProcAttr.Setpgid {
//	    pgid = pid                                      // ← both branches equal
//	}
//
// That `if` was a no-op, so the recorded PGID claimed a process group existed
// whether or not one had been created. A group is created only by Setpgid, and
// Setpgid is set only by Sandbox.Apply, which returns early when sandboxing is
// disabled (PermissiveConfig, or Sandbox.Enabled=false). Without it the child
// stays in the PARENT's process group, so no group with that ID exists, and
// `kill(-childpid)` fails with ESRCH. Measured directly:
//
//	child pid=2596613  actual pgid=2596503 (= the parent's)
//	syscall.Kill(-2596613, SIGKILL) -> ESRCH; child still alive
//
// The error was discarded, so nothing noticed, and `<-done` then waited for a
// process that had never been signalled. The timeout — the very mechanism meant
// to bound the call — silently did nothing.
//
// ─── DEFECT B: Wait() parks on the I/O goroutines a descendant holds open ───
//
// Execute assigns a NON-*os.File writer to Cmd.Stdout/Stderr (writerAdapter over
// the OutputCollector), so os/exec creates its own pipes and copy goroutines and
// Cmd.Wait waits for them (awaitGoroutines). A copy goroutine ends on EOF, and a
// pipe reaches EOF only once EVERY write end is closed — including the copies a
// GRANDCHILD inherited. Cmd.WaitDelay was not set, so that wait had no bound.
// Measured directly, with the direct child exiting immediately:
//
//	sh -c "sleep 8 & echo hello"  ->  Cmd.Wait() returned after 8.01s
//
// Unlike defect A this needs no unusual configuration; it needs only that the
// call cannot be cancelled, which is the case whenever Timeout is 0 and the
// context is not cancellable — reachable through the exported NewDefaultExecutor
// (which, unlike NewShellExecutor, does not run applyDefaults and so does not
// substitute the config's DefaultTimeout).
//
// ─── THE FIX BEING GUARDED ─────────────────────────────────────────────────
//
// A: record PGID 0 unless Setpgid actually ran, so Send falls back to signalling
//    the PID directly — the kill lands instead of returning ESRCH.
// B: set Cmd.WaitDelay, whose timer starts when Wait observes the process exit
//    and which on expiry closes the parent's pipe descriptors, releasing copy
//    goroutines that a descendant would otherwise hold forever.
//
// §11.4.115 RED-polarity switch: ONE test source, two roles.
//
//	RED_MODE=1 → reproduce each defect on the pre-fix artifact: assert the call
//	             does NOT return within the decision deadline. The wait is
//	             BOUNDED, so a failing run reports instead of wedging the suite.
//	RED_MODE=0 → standing GREEN guard (default): assert it DOES return promptly,
//	             with exit code and output preserved.
//
// §11.4.135: these are ordinary `go test` cases, so they run on every build —
// the same registration the HXC-184 and HXC-198 guards use. The four
// preserve-property cases assert unconditionally in BOTH polarities, so the new
// bound cannot silently become a truncator or a premature killer.

// redMode() (background_security_workdir_test.go) and grandchildLifetime /
// graceDecisionDeadline (stream_grandchild_grace_test.go) are shared with the
// sibling guards in this package and are reused here rather than redeclared, so
// all three defect families are judged against the same polarity switch and the
// same timing margins.

// executeResult carries an Execute return across the goroutine that bounds it.
type executeResult struct {
	res *ExecutionResult
	err error
}

// runExecuteBounded invokes Execute on its own goroutine and waits at most
// `within` for it to return.
//
// The goroutine is deliberately NOT reaped on timeout: on the pre-fix artifact
// it stays parked, which is precisely the leak being demonstrated. Every command
// used here self-terminates, so the test binary still exits.
func runExecuteBounded(
	ctx context.Context,
	run func(context.Context, *Command) (*ExecutionResult, error),
	cmd *Command,
	within time.Duration,
) (*executeResult, time.Duration) {
	done := make(chan executeResult, 1)
	start := time.Now()
	go func() {
		res, err := run(ctx, cmd)
		done <- executeResult{res: res, err: err}
	}()

	select {
	case r := <-done:
		return &r, time.Since(start)
	case <-time.After(within):
		return nil, time.Since(start)
	}
}

// pidFileCommand builds a command that records the PID of the process that will
// actually be running when the timeout fires, then blocks.
//
// `exec` is load-bearing: it replaces the shell image rather than forking, so
// the PID written to the file IS the direct child Execute registered. Without it
// the sleep would be a grandchild and the file would name a shell that had
// already moved on, which would prove nothing about whether the kill landed.
func pidFileCommand(t *testing.T, sleepFor string) (command, pidPath string) {
	t.Helper()
	pidPath = filepath.Join(t.TempDir(), "child.pid")
	return fmt.Sprintf("echo $$ > %s; exec sleep %s", pidPath, sleepFor), pidPath
}

// readPID waits briefly for the command to record its PID and returns it.
func readPID(t *testing.T, pidPath string) int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(pidPath)
		if err == nil {
			if pid, convErr := strconv.Atoi(strings.TrimSpace(string(raw))); convErr == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("command never recorded its PID at %s", pidPath)
	return 0
}

// processAlive reports whether pid still exists.
//
// signal 0 performs only the existence + permission check and delivers nothing
// (POSIX kill(2)). A reaped child would be a zombie and would still answer here,
// so this is only meaningful for a process THIS test did not wait on — which is
// the case for the sandbox-disabled child, whose reaping is what we are testing.
func processAlive(pid int) bool {
	return syscall.Kill(pid, syscall.Signal(0)) == nil
}

// permissiveExecutor returns an executor configured exactly as the exported
// PermissiveConfig does at the point that matters: sandboxing off, so Setpgid is
// never applied and no process group is created for the child.
func permissiveExecutor() *ShellExecutor {
	config := DefaultConfig()
	config.Sandbox.Enabled = false
	return NewShellExecutor(config)
}

// ── DEFECT A ────────────────────────────────────────────────────────────────

// TestExecuteTimeoutKillsTheChildWhenSandboxingIsDisabled is the HXC-211 defect-A
// guard: with no process group to signal, the timeout's SIGKILL must still reach
// the child, and Execute must return.
//
// The assertion is deliberately sink-side (§11.4.69): it is not enough that
// Execute returned — the CHILD PROCESS MUST BE GONE. A bound that returned while
// leaving the process running would be a different defect wearing this test's
// green.
func TestExecuteTimeoutKillsTheChildWhenSandboxingIsDisabled(t *testing.T) {
	command, pidPath := pidFileCommand(t, "30")

	se := permissiveExecutor()
	cmd := &Command{
		ID:      "hxc211-defect-a",
		Command: command,
		Timeout: 1 * time.Second,
	}

	// The child sleeps 30s. If the kill lands, Execute returns ~1s after start.
	// If it does not, Execute waits out the full 30s, so a 8s decision deadline
	// separates the two outcomes with a wide margin either way.
	const decisionDeadline = 8 * time.Second

	pidCh := make(chan int, 1)
	go func() { pidCh <- readPID(t, pidPath) }()

	r, elapsed := runExecuteBounded(context.Background(), se.Execute, cmd, decisionDeadline)

	if redMode() {
		require.Nil(t, r,
			"RED_MODE=1: Execute returned after %s — the SIGKILL reached the child, "+
				"so the defect is NOT present on this artifact", elapsed)
		t.Logf("RED confirmed: Execute still parked after %s despite a 1s timeout "+
			"(the kill went to a process group that does not exist)", elapsed)
		return
	}

	require.NotNil(t, r,
		"Execute did not return within %s despite a 1s timeout: the timeout's SIGKILL "+
			"went to a process group that was never created (kill(-pid) -> ESRCH) and the "+
			"discarded error hid it, so the wait for the child had no bound", decisionDeadline)
	require.NoError(t, r.err, "a killed command is reported via Killed, not as an error")
	require.NotNil(t, r.res)
	assert.True(t, r.res.Killed, "the result must record that the command was killed")
	assert.Less(t, elapsed, 5*time.Second,
		"Execute should return shortly after the 1s timeout, not wait out the child")

	// Sink-side proof: the process itself must be gone, not merely abandoned.
	pid := <-pidCh
	deadline := time.Now().Add(3 * time.Second)
	for processAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	assert.False(t, processAlive(pid),
		"pid %d survived the timeout: Execute returned but the SIGKILL never landed, "+
			"which leaks the process rather than bounding the call", pid)
}

// TestKillReachesAnUnsandboxedChild guards the same root cause through the
// exported Kill API rather than the timeout path, since both read the PGID that
// Execute registered.
func TestKillReachesAnUnsandboxedChild(t *testing.T) {
	command, pidPath := pidFileCommand(t, "30")

	se := permissiveExecutor()
	cmd := &Command{
		ID:      "hxc211-defect-a-kill",
		Command: command,
		Timeout: 25 * time.Second,
	}

	exec, err := se.ExecuteAsync(context.Background(), cmd)
	require.NoError(t, err)
	t.Cleanup(exec.Cancel)

	pid := readPID(t, pidPath)
	require.True(t, processAlive(pid), "child should be running before we signal it")

	killErr := se.Kill(cmd.ID, syscall.SIGKILL)

	if redMode() {
		// Pre-fix, the signal is aimed at a non-existent process group.
		require.Error(t, killErr,
			"RED_MODE=1: Kill succeeded — the PGID bookkeeping is already correct here")
		t.Logf("RED confirmed: Kill returned %v (signalled a group that was never created)", killErr)
		return
	}

	require.NoError(t, killErr,
		"Kill must reach a child that has no process group of its own; a PGID recorded "+
			"for a group that was never created makes kill(-pid) fail with ESRCH")

	deadline := time.Now().Add(3 * time.Second)
	for processAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	assert.False(t, processAlive(pid), "pid %d survived an explicit Kill", pid)
}

// TestExecuteStreamTimeoutKillsTheChildWhenSandboxingIsDisabled guards the
// THIRD instance of defect A, which the sweep that opened HXC-211 did not list.
//
// ExecuteStream's cancel branch has the identical shape as Execute's:
//
//	e.signalHandler.Send(cmd.ID, syscall.SIGKILL)
//	<-waitDone // Wait for process to actually exit
//
// and it reads the SAME registered PGID, written by the same vacuous `if`. The
// drain grace added by HXC-184 does not rescue it: the grace bounds the
// SCANNERS, and this wait sits BEFORE the drain loop, waiting on the process.
func TestExecuteStreamTimeoutKillsTheChildWhenSandboxingIsDisabled(t *testing.T) {
	command, pidPath := pidFileCommand(t, "30")

	se := permissiveExecutor()
	cmd := &Command{
		ID:      "hxc211-defect-a-stream",
		Command: command,
		Timeout: 1 * time.Second,
	}

	streaming, err := se.ExecuteStream(context.Background(), cmd)
	require.NoError(t, err)
	t.Cleanup(streaming.Cancel)

	// Drain concurrently, per the StreamingExecution contract.
	go func() {
		for range streaming.Stdout {
		}
	}()
	go func() {
		for range streaming.Stderr {
		}
	}()

	pid := readPID(t, pidPath)

	const decisionDeadline = 8 * time.Second
	start := time.Now()

	var result *ExecutionResult
	select {
	case result = <-streaming.Done:
	case <-time.After(decisionDeadline):
	}
	elapsed := time.Since(start)

	if redMode() {
		require.Nil(t, result,
			"RED_MODE=1: ExecuteStream returned after %s — the SIGKILL reached the child, "+
				"so the defect is NOT present on this artifact", elapsed)
		t.Logf("RED confirmed: ExecuteStream still parked after %s despite a 1s timeout", elapsed)
		return
	}

	require.NotNil(t, result,
		"ExecuteStream did not deliver a result within %s despite a 1s timeout: the "+
			"`<-waitDone` after the SIGKILL has no bound when the signal never lands",
		decisionDeadline)
	assert.True(t, result.Killed)

	deadline := time.Now().Add(3 * time.Second)
	for processAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	assert.False(t, processAlive(pid),
		"pid %d survived the streaming timeout: the result was delivered but the "+
			"SIGKILL never landed", pid)
}

// ── DEFECT B ────────────────────────────────────────────────────────────────

// TestExecuteIsBoundedWhenAGrandchildHoldsThePipe is the HXC-211 defect-B guard:
// with no timeout and a non-cancellable context, Execute must still return once
// the DIRECT child has exited, even though a grandchild keeps the pipe write
// ends open.
//
// NewDefaultExecutor is used deliberately rather than NewShellExecutor: it is the
// exported constructor documented in doc.go, and unlike NewShellExecutor it does
// not run applyDefaults, so Timeout stays 0 and no timeout is substituted.
func TestExecuteIsBoundedWhenAGrandchildHoldsThePipe(t *testing.T) {
	e := NewDefaultExecutor(DefaultConfig())
	cmd := &Command{
		ID:      "hxc211-defect-b",
		Command: fmt.Sprintf("sleep %d & echo started", int(grandchildLifetime.Seconds())),
		Timeout: 0, // no timeout, and the context below cannot be cancelled
	}

	// The direct child exits at once. Bounded, Execute returns after roughly one
	// streamDrainGrace (~2s); unbounded, it waits out grandchildLifetime (12s).
	// graceDecisionDeadline (6s) sits between the two with margin on both sides,
	// so the verdict is not a timing coin-flip on a loaded host.
	r, elapsed := runExecuteBounded(context.Background(), e.Execute, cmd, graceDecisionDeadline)

	if redMode() {
		require.Nil(t, r,
			"RED_MODE=1: Execute returned after %s — the drain is already bounded on "+
				"this artifact, so the defect is NOT present", elapsed)
		t.Logf("RED confirmed: Execute still parked after %s although the direct child "+
			"exited immediately (Cmd.Wait held by the grandchild's pipe copies)", elapsed)
		return
	}

	require.NotNil(t, r,
		"Execute did not return within %s. The direct child exited immediately; the wait "+
			"is held by os/exec's output-copy goroutines, which cannot see EOF while a "+
			"grandchild holds the pipe write ends. Nothing bounds that wait: there is no "+
			"timeout, the context is not cancellable, and Cmd.WaitDelay is unset",
		graceDecisionDeadline)
	require.NoError(t, r.err)
	require.NotNil(t, r.res)
	assert.Contains(t, r.res.Stdout, "started",
		"output produced before the bound fired must still be collected")
	assert.Less(t, elapsed, graceDecisionDeadline,
		"Execute must be bounded by the drain grace, not by the grandchild's lifetime")
	assert.True(t, r.res.OutputIncomplete,
		"a drain cut short by the bound is a truncation and must be disclosed, not "+
			"returned as output that looks complete")
}

// ── PRESERVE-PROPERTY GUARDS (assert in BOTH polarities) ────────────────────
//
// These four hold unconditionally. A bound that buys promptness by truncating
// healthy output, by killing commands early, or by losing exit codes would be a
// worse defect than the hang it replaced, so each of those is pinned here.

// TestExecuteWaitsForALongForegroundCommandWithNoTimeout is the anti-over-bound
// guard, and the most important test in this file.
//
// The new bound must start when the process EXITS, never when it starts. A
// 3-second foreground command with no timeout must run to completion and report
// success — if the bound were a total budget measured from Start, this command
// would be cut off after one drain grace and the "fix" would be a regression far
// worse than the hang.
func TestExecuteWaitsForALongForegroundCommandWithNoTimeout(t *testing.T) {
	e := NewDefaultExecutor(DefaultConfig())
	cmd := &Command{
		ID:      "hxc211-preserve-foreground",
		Command: "sleep 3; echo finished",
		Timeout: 0,
	}

	r, elapsed := runExecuteBounded(context.Background(), e.Execute, cmd, 20*time.Second)
	require.NotNil(t, r, "a plain foreground command must complete")
	require.NoError(t, r.err)
	require.NotNil(t, r.res)

	assert.Equal(t, 0, r.res.ExitCode)
	assert.Contains(t, r.res.Stdout, "finished",
		"the command ran to completion; its output must be intact")
	assert.False(t, r.res.OutputIncomplete,
		"a command that reached clean EOF must not be reported as truncated")
	assert.GreaterOrEqual(t, elapsed, 3*time.Second,
		"the command was not allowed to finish: the bound must measure from process "+
			"exit, not from process start")
}

// TestExecuteStillReturnsCompleteOutput pins that the bound does not truncate an
// ordinary command's output, including output large enough to fill a pipe buffer
// several times over.
func TestExecuteStillReturnsCompleteOutput(t *testing.T) {
	const lines = 5000

	e := NewDefaultExecutor(DefaultConfig())
	cmd := &Command{
		ID:      "hxc211-preserve-output",
		Command: fmt.Sprintf("i=1; while [ $i -le %d ]; do echo line-$i; i=$((i+1)); done", lines),
		Timeout: 30 * time.Second,
	}

	r, _ := runExecuteBounded(context.Background(), e.Execute, cmd, 60*time.Second)
	require.NotNil(t, r)
	require.NoError(t, r.err)
	require.NotNil(t, r.res)

	assert.False(t, r.res.OutputIncomplete, "a complete read must not be flagged truncated")
	assert.Contains(t, r.res.Stdout, "line-1")
	assert.Contains(t, r.res.Stdout, fmt.Sprintf("line-%d", lines),
		"the final line must survive: a bound that drops the tail of a healthy "+
			"command's output is a truncator, not a fix")
	assert.Equal(t, lines, strings.Count(r.res.Stdout, "line-"),
		"every line must be present exactly once")
}

// TestExecuteStillReportsNonZeroExitCode pins that the bound does not swallow or
// overwrite a genuine exit status. os/exec reports a WaitDelay expiry only when
// there is no other error to report, so an ExitError must continue to win.
func TestExecuteStillReportsNonZeroExitCode(t *testing.T) {
	e := NewDefaultExecutor(DefaultConfig())
	cmd := &Command{
		ID:      "hxc211-preserve-exitcode",
		Command: "echo out; exit 7",
		Timeout: 30 * time.Second,
	}

	r, _ := runExecuteBounded(context.Background(), e.Execute, cmd, 30*time.Second)
	require.NotNil(t, r)
	require.NotNil(t, r.res)

	assert.Equal(t, 7, r.res.ExitCode, "the command's real exit status must be preserved")
	assert.Nil(t, r.res.Error, "a non-zero exit is reported via ExitCode, not as Error")
	assert.Contains(t, r.res.Stdout, "out")
}

// TestExecuteSucceedsCleanlyWithSandboxDisabled pins that the PGID change does
// not disturb the ordinary path: a command that exits on its own under a
// permissive config must still be reaped and reported normally.
func TestExecuteSucceedsCleanlyWithSandboxDisabled(t *testing.T) {
	se := permissiveExecutor()
	cmd := &Command{
		ID:      "hxc211-preserve-permissive",
		Command: "echo permissive-ok",
		Timeout: 20 * time.Second,
	}

	r, _ := runExecuteBounded(context.Background(), se.Execute, cmd, 30*time.Second)
	require.NotNil(t, r)
	require.NoError(t, r.err)
	require.NotNil(t, r.res)

	assert.Equal(t, 0, r.res.ExitCode)
	assert.False(t, r.res.Killed, "a command that exited on its own was not killed")
	assert.Contains(t, r.res.Stdout, "permissive-ok")
}

// ── SITE 3: OutputStreamer's reader-ownership contract ──────────────────────
//
// output.go holds the same unbounded shape twice — the `for scanner.Scan()`
// loop in streamOutput, and the `wg.Wait()` in Start that closes Done. Neither
// is a defect there and neither is fixable there: OutputStreamer is handed plain
// io.Readers, which carry no Close, so it cannot release a scanner parked in
// Read without closing descriptors it does not own — which would double-close
// them under the one caller that DOES own them.
//
// The bound therefore lives with the reader owner. ExecuteStream is currently
// the only production caller (executor.go, NewOutputStreamer(pipes.stdoutR,
// pipes.stderrR)) and discharges it in a specific order:
//
//	streamer.Stop()
//	pipes.closeReadEnds()   // ← the load-bearing line
//	<-streamer.Done()       // ← unbounded, and safe ONLY because of the line above
//
// Drop or reorder that middle line and the wait below it becomes the HXC-198
// hang verbatim. So this is a latent trap rather than a live bug, and the
// treatment is to pin the contract rather than to change behaviour: the test
// below fails the moment Stop-alone becomes sufficient (someone gave the
// streamer ownership it should not have) or the moment closing the readers stops
// being sufficient (the release path broke).
//
// §11.4.115 note, stated plainly rather than faked: this guard has NO RED
// polarity, because there is no defect here to reproduce. It asserts a standing
// invariant that holds identically before and after HXC-211, so a RED_MODE
// branch would have nothing to assert. It is registered as a §11.4.135 standing
// guard on the strength of the invariant, not of a fixed defect.

// TestOutputStreamerDoneRequiresTheReaderOwnerToCloseTheReaders pins both halves
// of the contract in one run: Stop alone must NOT finish the streamer, and
// closing the readers MUST.
//
// The pipes' write ends are deliberately left open for the first half — that is
// exactly the shape a surviving grandchild produces, and it is what makes the
// negative assertion meaningful rather than a race: while a write end is open
// the scanners genuinely CANNOT reach EOF, so "Done did not close" is a
// determinism-safe property here and not a timing coin-flip (§11.4.201).
func TestOutputStreamerDoneRequiresTheReaderOwnerToCloseTheReaders(t *testing.T) {
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)
	defer stdoutW.Close()

	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)
	defer stderrW.Close()

	streamer := NewOutputStreamer(stdoutR, stderrR)
	streamer.Start()

	delivered := make(chan string, 4)
	go func() {
		for line := range streamer.GetStdout() {
			delivered <- line
		}
	}()
	go func() {
		for range streamer.GetStderr() {
		}
	}()

	// Establish that the streamer is genuinely live and reading. Without this,
	// "Done did not close" below could be explained by a streamer that never
	// started, which would prove nothing.
	_, err = fmt.Fprintln(stdoutW, "first")
	require.NoError(t, err)
	select {
	case got := <-delivered:
		require.Equal(t, "first", got, "the streamer must be delivering before we test teardown")
	case <-time.After(5 * time.Second):
		t.Fatal("the streamer never delivered its first line; the rest of this test would be vacuous")
	}

	// Half 1 — Stop is not a teardown. The write ends are still open, so both
	// scanners are parked in Read, where Stop cannot reach them.
	streamer.Stop()
	select {
	case <-streamer.Done():
		t.Fatal("Done closed after Stop alone, with the pipes' write ends still open. " +
			"Stop only releases a scanner parked on a channel SEND; one parked in Read " +
			"can be released only by closing the reader. If this now passes, the streamer " +
			"has taken ownership of readers it did not create — re-audit ExecuteStream's " +
			"closeReadEnds/Done ordering and check for a double close")
	case <-time.After(1500 * time.Millisecond):
		// Expected: nothing can finish the scanners yet.
	}

	// Half 2 — the reader OWNER closes, which is the only thing that can release
	// a scanner blocked in Read (os.File.Close wakes it with ErrClosed).
	require.NoError(t, stdoutR.Close())
	require.NoError(t, stderrR.Close())

	select {
	case <-streamer.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("Done did not close even after the readers were closed. Closing the read " +
			"ends is ExecuteStream's only release path for a scanner parked on a pipe a " +
			"grandchild still holds open; if it no longer works, the `<-streamer.Done()` " +
			"that follows closeReadEnds is once again an unbounded wait")
	}
}

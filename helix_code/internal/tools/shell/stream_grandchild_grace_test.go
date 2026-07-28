package shell

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/iotest"
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
	grandchildLifetime = 12 * time.Second

	// graceDecisionDeadline is the bounded window both polarities are judged
	// against.
	//
	// The FIXED artifact releases after at most TWO no-progress windows (~4s):
	// the command's own output is delivered during the first window, which
	// counts as progress and rearms the timer once; the second window sees
	// nothing and fires. It sits between that ~4s and grandchildLifetime (12s —
	// when even the BROKEN artifact would eventually release), leaving a 1.5x
	// margin below and a 2x margin above, so the verdict is not a timing
	// coin-flip on a loaded host.
	graceDecisionDeadline = 6 * time.Second
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

// TestExecuteStreamDoesNotTruncateASlowButDrainingConsumer is the regression
// guard for I1 — a defect introduced by the FIRST version of this fix and
// caught in review, not by me.
//
// That version treated the grace as a total drain budget, reasoning that only
// "<=64 KiB of already-buffered data, milliseconds of work" could remain after
// the child was reaped. The reasoning was wrong about WHOSE pace matters. A
// healthy command with no grandchild at all can exit instantly with everything
// already in the kernel buffer; the 100-slot channel then fills, the scanner
// parks on send, and a consumer following the documented concurrent-drain
// contract at a few ms/line is cut off mid-stream. Measured on that version:
// 490 of 5000 lines delivered, where the pre-fix code delivered all 5000.
//
// The fix is that the grace is a NO-PROGRESS window, rearmed while lines are
// still being delivered. This case pins that: the command exits immediately,
// there is no grandchild, and the consumer deliberately reads slowly enough to
// span several grace windows. Every line must still arrive.
//
// Asserted unconditionally in both polarities: completeness for a draining
// consumer is a property the fix must PRESERVE, so it must hold on the pre-fix
// artifact too — only the intermediate version failed it.
func TestExecuteStreamDoesNotTruncateASlowButDrainingConsumer(t *testing.T) {
	const (
		lineCount    = 500                   // >> the 100-slot channel buffer
		perLineDelay = 10 * time.Millisecond // ~5s total: spans ~2.5 grace windows
	)

	executor := NewShellExecutor(DefaultConfig())

	execution, err := executor.ExecuteStream(context.Background(), &Command{
		ID:      "hxc184-slow-but-draining-consumer",
		Command: fmt.Sprintf("seq 1 %d", lineCount),
	})
	require.NoError(t, err)

	// A consumer that honours the contract — draining concurrently — but slowly.
	type drained struct {
		count int
		last  string
	}
	got := make(chan drained, 1)
	go func() {
		var d drained
		for line := range execution.Stdout {
			d.last = line
			d.count++
			time.Sleep(perLineDelay)
		}
		got <- d
	}()

	var result *ExecutionResult
	select {
	case result = <-execution.Done:
	case <-time.After(60 * time.Second):
		t.Fatal("ExecuteStream did not complete within 60s for a slow-but-draining consumer")
	}

	d := <-got

	assert.Equal(t, lineCount, d.count,
		"a consumer that is still draining must never be cut off for being slow")
	assert.Equal(t, fmt.Sprint(lineCount), d.last, "the stream must run to its true end")
	assert.False(t, result.OutputIncomplete,
		"nothing was dropped, so the truncation flag must stay clear")
	assert.NoError(t, result.Error)
	assert.Equal(t, 0, result.ExitCode)
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

// TestOutputStreamerErrPrefersGenuineFailureOverTeardown is the standing guard
// for M2.
//
// Err() used to return stdoutErr unconditionally, so a grace-path
// ErrStreamStopped on stdout masked a real stderr failure — and because callers
// filter ErrStreamStopped as expected, that real failure was then dropped
// entirely rather than reported. Review measured that reverting the fix left the
// whole in-tree suite passing, so nothing detected it: this is that missing
// detector.
//
// Deterministic: stderr fails on its first Read; stdout has more lines than its
// 100-slot buffer with nobody draining, so its scanner cannot reach EOF and is
// released by Stop with ErrStreamStopped.
//
// Paired §1.1 mutation: restore the stdout-first Err() and this FAILS.
func TestOutputStreamerErrPrefersGenuineFailureOverTeardown(t *testing.T) {
	genuine := errors.New("stderr read exploded")

	var payload strings.Builder
	for i := 0; i < 250; i++ { // >> the 100-slot buffer
		fmt.Fprintf(&payload, "line-%d\n", i)
	}

	streamer := NewOutputStreamer(strings.NewReader(payload.String()), iotest.ErrReader(genuine))
	streamer.Start()

	// The genuine failure must be recorded BEFORE the stream is abandoned, which
	// is the real ordering: a scanner dies on its own early, and the teardown
	// happens later when the grace expires. Abandoning first would be a
	// different case entirely — an error racing the teardown is deliberately
	// ATTRIBUTED to it (see streamOutput), because the two are indistinguishable
	// once Stop has been requested.
	require.Eventually(t, func() bool { return streamer.Err() != nil },
		5*time.Second, time.Millisecond,
		"the stderr scanner must record its genuine failure before we abandon the stream")

	streamer.Stop() // abandon: stdout records ErrStreamStopped
	<-streamer.Done()

	require.ErrorIs(t, streamer.Err(), genuine,
		"a genuine stream failure must outrank the deliberate-teardown sentinel, "+
			"whichever stream each came from")
	assert.NotErrorIs(t, streamer.Err(), ErrStreamStopped,
		"reporting the teardown here would let callers filter it and lose the real failure")
}

// TestExecuteStreamSurfacesStderrFailureDespiteStdoutTeardown is the end-to-end
// half of the M2 guard: it proves the priority actually changes what a CALLER
// sees, not merely what Err() returns.
//
// A single >4096-byte stderr line trips bufio.ErrTooLong — a genuine failure —
// while a grandchild holds stdout open with nothing to say, so stdout's scanner
// is torn down by the grace. Pre-M2 the stdout teardown won, the executor
// filtered it as expected, and the result carried NO error at all despite a real
// stream failure having occurred.
//
// (The ErrTooLong behaviour itself is pre-existing and tracked separately; this
// guard depends on it only as a convenient real failure.)
func TestExecuteStreamSurfacesStderrFailureDespiteStdoutTeardown(t *testing.T) {
	executor := NewShellExecutor(DefaultConfig())

	execution, err := executor.ExecuteStream(context.Background(), &Command{
		ID: "hxc184-stderr-failure-vs-stdout-teardown",
		Command: fmt.Sprintf(
			`sleep %d & head -c 5000 /dev/zero | tr '\0' 'x' >&2`,
			int(grandchildLifetime.Seconds())),
	})
	require.NoError(t, err)

	go func() {
		for range execution.Stdout { //nolint:revive // drained; stdout stays silent here
		}
	}()

	var result *ExecutionResult
	select {
	case result = <-execution.Done:
	case <-time.After(graceDecisionDeadline):
		t.Fatal("the grace must release the execution even with a failed stderr scanner")
	}

	require.Error(t, result.Error,
		"a genuine stderr failure must reach the caller even though stdout was "+
			"deliberately torn down by the grace")
	assert.ErrorIs(t, result.Error, bufio.ErrTooLong)
	assert.True(t, result.OutputIncomplete, "neither stream reached a clean EOF")
}

// TestExecuteStreamCancelMidStreamMarksOutputIncomplete is the standing guard
// for the first half of M3: Cancel drops in-flight lines via Stop, and that must
// be disclosed rather than reported as a complete stream.
//
// Deterministic: the test waits until the stdout buffer is FULL before
// cancelling, which means the scanner is provably parked on a send with lines
// still to come — cancelling earlier could race a clean EOF.
func TestExecuteStreamCancelMidStreamMarksOutputIncomplete(t *testing.T) {
	executor := NewShellExecutor(DefaultConfig())

	execution, err := executor.ExecuteStream(context.Background(), &Command{
		ID: "hxc184-cancel-mid-stream",
		// Emit far more than the buffer holds, then linger so the process is
		// still alive and killable when Cancel lands.
		Command: fmt.Sprintf("seq 1 500; sleep %d", int(grandchildLifetime.Seconds())),
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool { return len(execution.Stdout) == 100 },
		graceDecisionDeadline, 5*time.Millisecond,
		"the stdout buffer must fill so the scanner is parked with lines still in flight")

	execution.Cancel()

	select {
	case result := <-execution.Done:
		assert.True(t, result.Killed, "Cancel kills the process")
		assert.True(t, result.OutputIncomplete,
			"lines were still in flight when Cancel dropped them; that must not be silent")
		assert.NoError(t, result.Error,
			"a caller-requested cancel is ErrStreamStopped-class, not a stream failure")
	case <-time.After(graceDecisionDeadline):
		t.Fatal("Cancel must release the execution promptly")
	}
}

// TestExecuteStreamCancelAfterCleanEOFKeepsOutputComplete is the second half of
// the M3 guard, and the sharper direction: the flag must be honest when nothing
// was lost, or it becomes noise nobody trusts.
//
// The command emits its output, CLOSES both streams (so the scanners reach a
// genuine EOF), then lingers so it is still killable. Draining stdout to closure
// is the synchronisation point: the channels are closed only after BOTH scanners
// have returned, so by the time Cancel lands both streams are provably at EOF.
func TestExecuteStreamCancelAfterCleanEOFKeepsOutputComplete(t *testing.T) {
	executor := NewShellExecutor(DefaultConfig())

	execution, err := executor.ExecuteStream(context.Background(), &Command{
		ID: "hxc184-cancel-after-clean-eof",
		Command: fmt.Sprintf("echo hi; exec sleep %d >/dev/null 2>/dev/null",
			int(grandchildLifetime.Seconds())),
	})
	require.NoError(t, err)

	var lines []string
	for line := range execution.Stdout {
		lines = append(lines, line)
	}
	require.Equal(t, []string{"hi"}, lines)

	execution.Cancel()

	select {
	case result := <-execution.Done:
		assert.True(t, result.Killed, "the process was still alive and was killed")
		assert.False(t, result.OutputIncomplete,
			"both streams reached a clean EOF before Cancel, so nothing was lost "+
				"and the flag must stay clear")
	case <-time.After(graceDecisionDeadline):
		t.Fatal("Cancel must release the execution promptly")
	}
}

// TestExecuteStreamAbandonedConsumerStaysBoundedWhileProducerTrickles pins the
// TRUE bound for an abandoned consumer, correcting a claim I got wrong.
//
// I originally documented that an abandoned consumer is torn down "within two
// windows", reasoning that its channel fills and delivery stops. Review broke
// that with a probe: a descendant trickling 1 line/s keeps landing lines in the
// 200 slots of combined headroom, and each landing rearms the timer — so a send
// completing means "delivered OR buffered", not "a consumer is reading". The
// measured instance ran 15.05s with no consumer at all; the worst case is
// roughly (headroom + 1) x grace.
//
// This guard asserts what is actually true and load-bearing: the execution
// EXCEEDS one grace window (so nobody re-asserts the two-window claim) yet still
// terminates on its own (so HXC-184 does not recur through this path).
func TestExecuteStreamAbandonedConsumerStaysBoundedWhileProducerTrickles(t *testing.T) {
	const trickleSeconds = 6

	executor := NewShellExecutor(DefaultConfig())

	execution, err := executor.ExecuteStream(context.Background(), &Command{
		ID: "hxc184-abandoned-consumer-trickling-producer",
		// The grandchild outlives the direct child and emits one line per
		// second — few enough to fit the headroom the whole way.
		Command: fmt.Sprintf("for i in $(seq 1 %d); do echo $i; sleep 1; done & echo started",
			trickleSeconds),
	})
	require.NoError(t, err)

	// Deliberately abandon the channels: never drain, never Cancel.
	start := time.Now()
	select {
	case <-execution.Done:
	case <-time.After(60 * time.Second):
		t.Fatal("an abandoned consumer must still terminate — the bound is long, not infinite")
	}
	elapsed := time.Since(start)

	assert.Greater(t, elapsed, streamDrainGrace,
		"buffered lines rearm the timer, so this case genuinely outlives a single "+
			"grace window — the 'bounded by one window' reading is wrong")
	t.Logf("abandoned consumer released after %s (grace=%s, ~%ds of trickled output)",
		elapsed, streamDrainGrace, trickleSeconds)
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

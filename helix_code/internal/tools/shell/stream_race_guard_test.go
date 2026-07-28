package shell

import (
	"bufio"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamPipesSurviveCmdWait is the DETERMINISTIC regression guard for the
// defect in which ExecuteStream yielded zero stdout lines while reporting a
// correct exit code 0.
//
// Why the guard is deterministic, with no timing dependence whatsoever:
//
// The production defect was a race between Cmd.Wait and the scanner goroutines.
// This test does not try to win that race — it removes the race entirely by
// making the losing interleaving an explicit, sequential statement order in the
// test body: Wait runs to completion on line N, the first read happens on line
// N+k. There are no sleeps, no goroutines, no scheduler dependence, so the
// outcome is identical on every machine at every load level.
//
// Against the PRE-FIX wiring (execCmd.StdoutPipe()), the reader is registered
// in Cmd.parentIOPipes and closed by Cmd.Wait via closeDescriptors
// (go1.26 os/exec/exec.go:954) before this test reads a single byte. Every read
// then fails with os.ErrClosed, so the assertions below fail 100% of the time.
//
// Against the FIXED wiring (parent-owned os.Pipe assigned to Cmd.Stdout), the
// reader is not known to os/exec at all, so Wait cannot touch it and the bytes
// the child already wrote are still sitting in the kernel pipe buffer. The
// assertions pass 100% of the time.
//
// Paired mutation (§1.1): revert newStreamPipes to use execCmd.StdoutPipe() /
// execCmd.StderrPipe() and this guard FAILS deterministically.
func TestStreamPipesSurviveCmdWait(t *testing.T) {
	execCmd := exec.Command("/bin/sh", "-c", "for i in 1 2 3 4 5; do echo $i; done")

	pipes, err := newStreamPipes(execCmd)
	require.NoError(t, err)
	defer pipes.closeAll()

	require.NoError(t, execCmd.Start())

	// Mirror production: the parent must drop its write ends after Start, or
	// the reader below would never see EOF and this test would hang.
	pipes.closeWriteEnds()

	// The ordering that used to lose all output, forced rather than raced:
	// the process is fully reaped BEFORE the first read is attempted.
	require.NoError(t, execCmd.Wait())
	require.Equal(t, 0, execCmd.ProcessState.ExitCode())

	var lines []string
	scanner := bufio.NewScanner(pipes.stdoutR)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	require.NoError(t, scanner.Err(),
		"the stream reader must remain valid after Cmd.Wait has reaped the process")
	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, lines,
		"output written before the process exited must still be readable after Wait")
}

// TestOutputStreamerSurfacesReadError is the deterministic guard for the second
// half of the defect: the reason a torn-down pipe presented as a clean,
// perfectly green "command produced no output".
//
// bufio.Scanner signals a hard read failure exactly the way it signals a clean
// EOF — Scan returns false — so discarding scanner.Err() makes an I/O failure
// indistinguishable from success with empty output. This guard feeds the
// streamer a reader that fails immediately and asserts the error is reported.
//
// Deterministic: iotest.ErrReader fails on the first Read, with no concurrency
// involved in producing the error.
//
// Paired mutation (§1.1): delete the scanner.Err() recording in
// OutputStreamer.streamOutput and this guard FAILS deterministically.
func TestOutputStreamerSurfacesReadError(t *testing.T) {
	readFailure := errors.New("simulated pipe teardown")

	streamer := NewOutputStreamer(iotest.ErrReader(readFailure), strings.NewReader(""))
	streamer.Start()

	var lines []string
	for line := range streamer.GetStdout() {
		lines = append(lines, line)
	}
	<-streamer.Done()

	assert.Empty(t, lines)
	require.Error(t, streamer.Err(),
		"a failed read must not be reported as a clean end of stream")
	assert.ErrorIs(t, streamer.Err(), readFailure)
}

// TestExecuteStreamDeliversAllOutputBeforeDone guards the end-to-end ordering
// invariant the fix establishes: the result is published only after the
// scanners have drained the pipes, so observing Done means all output has
// already been delivered rather than possibly still being in flight.
//
// The consumer here deliberately reads Done FIRST and only then drains stdout.
// Post-fix this is safe by construction: the result goroutine blocks on
// <-streamer.Done() before publishing, and five lines fit the 100-slot buffered
// channel, so no backpressure can arise.
//
// Honest scope note (§11.4.6): unlike the two guards above, this one's failure
// against the pre-fix code is highly probable but not provably certain — the
// pre-fix scanner could still win the race and buffer the lines. The
// deterministic proof is TestStreamPipesSurviveCmdWait; this guard's value is
// pinning the post-fix ordering contract so a future change cannot publish the
// result before the output.
func TestExecuteStreamDeliversAllOutputBeforeDone(t *testing.T) {
	executor := NewShellExecutor(DefaultConfig())

	cmd := &Command{
		ID:      "stream-ordering-guard",
		Command: "for i in 1 2 3 4 5; do echo $i; done",
	}

	execution, err := executor.ExecuteStream(context.Background(), cmd)
	require.NoError(t, err)

	result := <-execution.Done
	require.NoError(t, result.Error)
	assert.Equal(t, 0, result.ExitCode)

	var lines []string
	for line := range execution.Stdout {
		lines = append(lines, line)
	}

	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, lines,
		"every line must already be buffered by the time Done fires")
}

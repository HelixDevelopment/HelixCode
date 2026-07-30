package shell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// shellLineSink is an alias for the line-sink callback used by ExecuteWithProgress.
// It intentionally mirrors tools.LineSink without importing the parent package
// (dev.helix.code/internal/tools imports dev.helix.code/internal/tools/shell, so
// the reverse import would create a cycle).  The types_background.go layer in
// internal/tools bridges the two at the registry level.
type shellLineSink = func(string)

// ExecuteWithProgress runs the shell command and streams stdout/stderr lines
// through sink as they are produced. The final aggregated output and exit code
// are returned for compatibility with the existing shell tool's Execute return
// shape.
//
// This makes ShellExecutor implement tools.BackgroundAware (via the adapter in
// internal/tools/shell_tools.go). When ToolRegistry dispatches with
// run_in_background:true, the BackgroundManager invokes this method instead of
// Execute, letting the BackgroundTask's output ring receive real-time progress.
//
// SINK CONTRACT — sink must not block indefinitely.
//
// sink is invoked SYNCHRONOUSLY on the scanner goroutines, so it sits directly
// in the read path. The bounded drain below releases a scanner parked in Read
// (by closing the read ends), which is what makes a descendant holding the pipe
// open survivable — but it cannot release a scanner parked INSIDE sink, because
// the call has already been handed over. A sink that never returns therefore
// still parks this function, and no timeout here can change that.
//
// The production sink is BackgroundTask.AppendOutput, a mutex-guarded bounded
// ring that always returns, so this is a contract for future callers rather
// than a live exposure. A sink that may block should hand off to its own
// goroutine and bound the handoff itself.
func (se *ShellExecutor) ExecuteWithProgress(
	ctx context.Context,
	params map[string]interface{},
	sink shellLineSink,
) (interface{}, error) {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("shell: command must be a non-empty string")
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if cwd, ok := params["cwd"].(string); ok && cwd != "" {
		cmd.Dir = cwd
	}

	// Parent-owned pipes, exactly as ExecuteStream uses them (see newStreamPipes
	// in executor.go for the full rationale). cmd.StdoutPipe()/StderrPipe() would
	// hand the read ends to os/exec, which records them in Cmd.parentIOPipes and
	// closes them inside cmd.Wait(). That is incompatible with the bounded wait
	// below, which MUST reap the child before the scanners have finished: os/exec
	// would tear the readers down mid-stream and truncate. Owning the descriptors
	// ourselves also makes closeReadEnds() available as the only release valve for
	// a reader parked in Read on a pipe a grandchild still holds open.
	pipes, err := newStreamPipes(cmd)
	if err != nil {
		return nil, fmt.Errorf("shell: pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		pipes.closeAll()
		return nil, fmt.Errorf("shell: start: %w", err)
	}
	// Drop the parent's copies of the write ends now that the child has inherited
	// them. While the parent still holds one, the pipe always has a live writer
	// and the reader never observes EOF even after the child exits.
	pipes.closeWriteEnds()
	defer pipes.closeAll()

	var (
		mu sync.Mutex
		// progress counts lines handed onward. It is what distinguishes a stream
		// that is DRAINING (slowly) from one that is WEDGED, so the grace below
		// can be a no-progress window rather than a total budget.
		progress atomic.Uint64
		lines    []string
		appendLn = func(line string) {
			mu.Lock()
			lines = append(lines, line)
			mu.Unlock()
			progress.Add(1)
			if sink != nil {
				sink(line)
			}
		}
	)

	scan := func(rd io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		s := bufio.NewScanner(rd)
		s.Buffer(make([]byte, 4096), 1024*1024)
		for s.Scan() {
			appendLn(s.Text())
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go scan(pipes.stdoutR, &wg)
	go scan(pipes.stderrR, &wg)

	// Reap the DIRECT child FIRST, then drain — the ordering HXC-198 inverted.
	//
	// Waiting for the scanners first (the previous `wg.Wait()` here) is unbounded:
	// a scanner finishes on EOF, and a pipe reaches EOF only once EVERY write end
	// is closed, INCLUDING the copies a GRANDCHILD inherited. `sleep 300 & echo
	// started`, or any daemonising command, leaves those open long after the
	// direct child is reaped, so that wait parked indefinitely. Nothing rescued
	// it: the read ends were closed by cmd.Wait(), which sat behind the stuck
	// wait, and exec.CommandContext's cancellation only SIGKILLs the direct child
	// — already exited — while the grandchild keeps the descriptors. No time
	// limit and no give-up path, so the caller never returned and the background
	// task's in-flight slot was never released (HXC-198, the sibling of HXC-184).
	waitErr := cmd.Wait()

	scannersDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(scannersDone)
	}()

	// Give the scanners a BOUNDED grace to drain, then tear the readers down.
	// This mirrors exec.Cmd.WaitDelay, whose timer likewise starts when Wait
	// observes the process exit and which on expiry calls
	// closeDescriptors(c.parentIOPipes) to release reads still parked.
	//
	// The grace is a NO-PROGRESS window, not a total budget: a sink that consumes
	// slowly is DRAINING, not wedged, and cutting it off would lose output a
	// perfectly healthy command had already produced. So the timer is rearmed
	// whenever a line has been delivered since the last check, and only a FULL
	// window with nothing delivered counts as stuck. A silent descendant — the
	// HXC-198 shape — delivers nothing, so the first window fires.
	outputIncomplete := false
	drainDeadline := time.NewTimer(streamDrainGrace)
	defer drainDeadline.Stop()
	lastProgress := progress.Load()

drain:
	for {
		select {
		case <-scannersDone:
			// Clean EOF on both streams: every byte has been delivered.
			break drain

		case <-drainDeadline.C:
			if p := progress.Load(); p != lastProgress {
				lastProgress = p
				drainDeadline.Reset(streamDrainGrace)
				continue
			}
			// scannersDone can become ready alongside the timer, and select picks
			// a ready case at random. Re-check before tearing anything down so a
			// fully-delivered run is never reported as truncated.
			select {
			case <-scannersDone:
				break drain
			default:
			}
			// Closing the read ends is the load-bearing step: it is the only way
			// to release a scanner parked in Read on a pipe whose write end a
			// grandchild still holds (os.File.Close is safe against a concurrent
			// Read — the runtime poller wakes the reader with ErrClosed).
			pipes.closeReadEnds()
			<-scannersDone
			outputIncomplete = true
			break drain
		}
	}

	mu.Lock()
	output := strings.Join(lines, "\n")
	mu.Unlock()

	// Bounded truncation is a strictly better failure mode than an unbounded
	// hang, but it is still a truncation — so disclose it rather than returning a
	// partial output that looks complete. It is deliberately NOT an error: the
	// command itself may have succeeded, and reporting a healthy exit as a
	// failure would misclassify it (same reasoning as ExecutionResult.
	// OutputIncomplete, which ExecuteStream sets for the identical situation).
	if waitErr != nil {
		return map[string]interface{}{
			"output":            output,
			"exit_code":         exitCodeFromError(waitErr, cmd),
			"output_incomplete": outputIncomplete,
		}, fmt.Errorf("shell: command exit: %w", waitErr)
	}
	return map[string]interface{}{
		"output":            output,
		"exit_code":         0,
		"output_incomplete": outputIncomplete,
	}, nil
}

// exitCodeFromError extracts the OS exit code from a process error.
func exitCodeFromError(err error, cmd *exec.Cmd) int {
	if cmd != nil && cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return -1
}

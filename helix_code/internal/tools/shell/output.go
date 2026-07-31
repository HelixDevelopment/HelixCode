package shell

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// ErrStreamStopped reports that streaming was abandoned via Stop before the
// underlying reader reached EOF. It is an expected condition on the
// cancel/timeout path, so callers should distinguish it from a genuine read
// failure with errors.Is.
var ErrStreamStopped = errors.New("output streaming stopped before EOF")

// OutputStreamer streams command output in real-time.
//
// Reader ownership — read this before wiring a new source to it. An
// OutputStreamer NEVER closes the readers it was handed. It holds them as plain
// io.Readers, which carry no Close, so those descriptors belong to whoever
// created them and only that owner can tear them down.
//
// That makes the owner responsible for the streamer's TERMINATION, not merely
// for its own cleanup. Both scanners end only when their reader reports EOF or
// an error, and a pipe reports EOF only once every write end is closed —
// including the copies a GRANDCHILD inherited. So against a reader that can
// stay open indefinitely the scanners park in Read, Done never closes, and the
// output channels are never closed. Stop does not rescue that (see Stop).
//
// Both of this file's waits are therefore unbounded BY CONSTRUCTION — the
// scanner loop in streamOutput, and the wg.Wait in Start that closes Done — and
// that is not a defect to fix here: this type cannot bound them without closing
// descriptors it does not own, which would double-close them under the one
// caller that does own them.
//
// ExecuteStream discharges the contract by owning its pipes (streamPipes) and
// calling closeReadEnds BEFORE it waits on Done. A caller that instead wires
// this to cmd.StdoutPipe() and waits on Done with no teardown of its own
// reproduces HXC-198 verbatim.
type OutputStreamer struct {
	stdout      io.Reader
	stderr      io.Reader
	stdoutChan  chan string
	stderrChan  chan string
	maxLineSize int
	done        chan struct{}
	stop        chan struct{}
	stopOnce    sync.Once
	startOnce   sync.Once
	progress    atomic.Uint64

	mu        sync.Mutex
	stdoutErr error
	stderrErr error
}

// NewOutputStreamer creates a new output streamer
func NewOutputStreamer(stdout, stderr io.Reader) *OutputStreamer {
	return &OutputStreamer{
		stdout:      stdout,
		stderr:      stderr,
		stdoutChan:  make(chan string, 100),
		stderrChan:  make(chan string, 100),
		maxLineSize: 4096,
		done:        make(chan struct{}),
		stop:        make(chan struct{}),
	}
}

// Start starts streaming output. It is idempotent: repeated calls do not spawn
// additional scanners (which would double-close the output channels).
func (os *OutputStreamer) Start() {
	os.startOnce.Do(func() {
		var wg sync.WaitGroup

		wg.Add(2)
		go func() {
			defer wg.Done()
			os.streamOutput(os.stdout, os.stdoutChan, &os.stdoutErr)
		}()

		go func() {
			defer wg.Done()
			os.streamOutput(os.stderr, os.stderrChan, &os.stderrErr)
		}()

		go func() {
			wg.Wait()
			close(os.stdoutChan)
			close(os.stderrChan)
			close(os.done)
		}()
	})
}

// Stop abandons streaming, unblocking scanners that are parked trying to hand a
// line to a consumer that has stopped reading. Without it a caller that walks
// away from the output channels would wedge the scanners forever, since done is
// only closed once both scanners have returned.
//
// Stop is HALF of a teardown, never the whole of one. It cannot interrupt a
// scanner parked in Read on the reader itself: a blocked channel send and a
// blocked read are different waits, and closing os.stop only releases the
// former. Releasing the latter requires closing the reader, which only its
// owner can do (see the type doc). Calling Stop and then waiting on Done, with
// nothing closing the readers, is an unbounded wait.
func (os *OutputStreamer) Stop() {
	os.stopOnce.Do(func() {
		close(os.stop)
	})
}

// streamOutput streams output from a reader to a channel
func (os *OutputStreamer) streamOutput(reader io.Reader, ch chan<- string, errOut *error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, os.maxLineSize), os.maxLineSize)

	for scanner.Scan() {
		line := scanner.Text()
		select {
		case ch <- line:
			os.progress.Add(1)
		case <-os.stop:
			os.setErr(errOut, ErrStreamStopped)
			return
		}
	}

	// A scanner error is NOT a clean EOF. Recording it is what stops a
	// truncated or failed read from masquerading as "the command produced no
	// output": bufio.Scanner reports both as a plain end of iteration, so
	// discarding scanner.Err() turns a hard I/O failure into a silent,
	// perfectly green empty result.
	if err := scanner.Err(); err != nil {
		select {
		case <-os.stop:
			// Streaming was deliberately abandoned — the caller cancelled, or
			// the reader was torn down when the drain grace expired with a
			// descendant still holding the pipe write ends open. The read failed
			// because WE closed it, so it is classified as the expected
			// ErrStreamStopped rather than as a genuine I/O defect. The
			// underlying cause stays wrapped and reachable via errors.Is/As.
			//
			// Honest scope note (§11.4.6): once Stop has been requested a real
			// I/O failure racing the teardown is indistinguishable from the
			// teardown itself, so it is absorbed here. exec.Cmd.awaitGoroutines
			// takes the same position, discarding the error outright on the
			// WaitDelay path; this at least preserves it.
			os.setErr(errOut, fmt.Errorf("%w: %w", ErrStreamStopped, err))
		default:
			os.setErr(errOut, err)
		}
	}
}

// setErr records the first error observed on a stream.
func (os *OutputStreamer) setErr(errOut *error, err error) {
	os.mu.Lock()
	defer os.mu.Unlock()
	if *errOut == nil {
		*errOut = err
	}
}

// Progress reports how many lines have been accepted by the output channels
// across both streams. It is monotonic, so callers can compare two samples.
//
// It advances on a COMPLETED SEND, which is not the same as end-to-end
// delivery: a send completes either because a consumer received the line or
// because there was still buffer headroom to absorb it. So an idle consumer
// still shows progress until its channel fills — only once the buffers are
// full does an absent consumer stop advancing this.
//
// What it does reliably distinguish is "lines are still moving" from "nothing
// is moving at all", which is what ExecuteStream needs to tell a slow-but-live
// stream apart from a stalled one. It is NOT a proof that a consumer exists.
func (os *OutputStreamer) Progress() uint64 {
	return os.progress.Load()
}

// Err returns the most significant error observed while streaming, or nil if
// both streams were read cleanly to EOF. It is only meaningful once Done is
// closed.
//
// A genuine failure outranks the ErrStreamStopped teardown sentinel regardless
// of which stream it came from. Returning stdout's error unconditionally would
// let a deliberate stdout teardown mask a REAL stderr failure (e.g.
// bufio.ErrTooLong) — and because callers filter ErrStreamStopped as expected,
// that genuine failure would then be dropped entirely rather than reported.
func (os *OutputStreamer) Err() error {
	os.mu.Lock()
	defer os.mu.Unlock()

	for _, err := range [2]error{os.stdoutErr, os.stderrErr} {
		if err != nil && !errors.Is(err, ErrStreamStopped) {
			return err
		}
	}
	if os.stdoutErr != nil {
		return os.stdoutErr
	}
	return os.stderrErr
}

// GetStdout returns the stdout channel
func (os *OutputStreamer) GetStdout() <-chan string {
	return os.stdoutChan
}

// GetStderr returns the stderr channel
func (os *OutputStreamer) GetStderr() <-chan string {
	return os.stderrChan
}

// Done returns a channel that is closed once BOTH scanners have returned and
// the output channels have been closed.
//
// It is not self-bounding. A scanner returns on EOF or on a read error, so a
// reader that reaches neither holds this open indefinitely. Waiting on it is
// bounded only AFTER the reader owner has closed the readers; waiting on it
// before that is the unbounded wait the type doc describes.
func (os *OutputStreamer) Done() <-chan struct{} {
	return os.done
}

// OutputCollector collects command output with size limits
type OutputCollector struct {
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
	maxSize     int64
	currentSize atomic.Int64
	truncated   atomic.Bool
	mu          sync.Mutex
}

// NewOutputCollector creates a new output collector
func NewOutputCollector(maxSize int64) *OutputCollector {
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // 10 MB default
	}
	return &OutputCollector{
		stdout:  &bytes.Buffer{},
		stderr:  &bytes.Buffer{},
		maxSize: maxSize,
	}
}

// WriteStdout writes to stdout buffer
func (oc *OutputCollector) WriteStdout(p []byte) (n int, err error) {
	return oc.write(oc.stdout, p)
}

// WriteStderr writes to stderr buffer
func (oc *OutputCollector) WriteStderr(p []byte) (n int, err error) {
	return oc.write(oc.stderr, p)
}

// write writes data to a buffer with size limit
func (oc *OutputCollector) write(buf *bytes.Buffer, p []byte) (int, error) {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	if oc.truncated.Load() {
		return len(p), nil // Discard if already truncated
	}

	newSize := oc.currentSize.Load() + int64(len(p))
	if newSize > oc.maxSize {
		oc.truncated.Store(true)
		remaining := oc.maxSize - oc.currentSize.Load()
		if remaining > 0 {
			buf.Write(p[:remaining])
			buf.WriteString("\n... [output truncated] ...\n")
		}
		return len(p), nil
	}

	n, err := buf.Write(p)
	oc.currentSize.Add(int64(n))
	return n, err
}

// GetOutput returns collected output
func (oc *OutputCollector) GetOutput() (stdout, stderr string, truncated bool) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	return oc.stdout.String(), oc.stderr.String(), oc.truncated.Load()
}

// Size returns the total size of collected output
func (oc *OutputCollector) Size() int64 {
	return oc.currentSize.Load()
}

// IsTruncated returns whether output was truncated
func (oc *OutputCollector) IsTruncated() bool {
	return oc.truncated.Load()
}

// Reset resets the collector to initial state
func (oc *OutputCollector) Reset() {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.stdout.Reset()
	oc.stderr.Reset()
	oc.currentSize.Store(0)
	oc.truncated.Store(false)
}

// writerAdapter adapts a write function to io.Writer
type writerAdapter struct {
	write func([]byte) (int, error)
}

func (w *writerAdapter) Write(p []byte) (int, error) {
	return w.write(p)
}

// CombinedOutputCollector collects stdout and stderr into a single stream
type CombinedOutputCollector struct {
	buffer      *bytes.Buffer
	maxSize     int64
	currentSize atomic.Int64
	truncated   atomic.Bool
	mu          sync.Mutex
}

// NewCombinedOutputCollector creates a new combined output collector
func NewCombinedOutputCollector(maxSize int64) *CombinedOutputCollector {
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // 10 MB default
	}
	return &CombinedOutputCollector{
		buffer:  &bytes.Buffer{},
		maxSize: maxSize,
	}
}

// Write writes data to the combined buffer
func (coc *CombinedOutputCollector) Write(p []byte) (int, error) {
	coc.mu.Lock()
	defer coc.mu.Unlock()

	if coc.truncated.Load() {
		return len(p), nil // Discard if already truncated
	}

	newSize := coc.currentSize.Load() + int64(len(p))
	if newSize > coc.maxSize {
		coc.truncated.Store(true)
		remaining := coc.maxSize - coc.currentSize.Load()
		if remaining > 0 {
			coc.buffer.Write(p[:remaining])
			coc.buffer.WriteString("\n... [output truncated] ...\n")
		}
		return len(p), nil
	}

	n, err := coc.buffer.Write(p)
	coc.currentSize.Add(int64(n))
	return n, err
}

// GetOutput returns the combined output
func (coc *CombinedOutputCollector) GetOutput() (output string, truncated bool) {
	coc.mu.Lock()
	defer coc.mu.Unlock()
	return coc.buffer.String(), coc.truncated.Load()
}

// Size returns the total size of collected output
func (coc *CombinedOutputCollector) Size() int64 {
	return coc.currentSize.Load()
}

// IsTruncated returns whether output was truncated
func (coc *CombinedOutputCollector) IsTruncated() bool {
	return coc.truncated.Load()
}

// Reset resets the collector to initial state
func (coc *CombinedOutputCollector) Reset() {
	coc.mu.Lock()
	defer coc.mu.Unlock()
	coc.buffer.Reset()
	coc.currentSize.Store(0)
	coc.truncated.Store(false)
}

// TeeWriter creates a writer that writes to multiple writers
type TeeWriter struct {
	writers []io.Writer
}

// NewTeeWriter creates a new tee writer
func NewTeeWriter(writers ...io.Writer) *TeeWriter {
	return &TeeWriter{
		writers: writers,
	}
}

// Write writes to all writers
func (tw *TeeWriter) Write(p []byte) (n int, err error) {
	for _, w := range tw.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
	return len(p), nil
}

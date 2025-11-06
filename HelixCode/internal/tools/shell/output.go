package shell

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"sync/atomic"
)

// OutputStreamer streams command output in real-time
type OutputStreamer struct {
	stdout      io.Reader
	stderr      io.Reader
	stdoutChan  chan string
	stderrChan  chan string
	maxLineSize int
	done        chan struct{}
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
	}
}

// Start starts streaming output
func (os *OutputStreamer) Start() {
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		os.streamOutput(os.stdout, os.stdoutChan)
	}()

	go func() {
		defer wg.Done()
		os.streamOutput(os.stderr, os.stderrChan)
	}()

	go func() {
		wg.Wait()
		close(os.stdoutChan)
		close(os.stderrChan)
		close(os.done)
	}()
}

// streamOutput streams output from a reader to a channel
func (os *OutputStreamer) streamOutput(reader io.Reader, ch chan<- string) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, os.maxLineSize), os.maxLineSize)

	for scanner.Scan() {
		line := scanner.Text()
		select {
		case ch <- line:
		case <-os.done:
			return
		}
	}
}

// GetStdout returns the stdout channel
func (os *OutputStreamer) GetStdout() <-chan string {
	return os.stdoutChan
}

// GetStderr returns the stderr channel
func (os *OutputStreamer) GetStderr() <-chan string {
	return os.stderrChan
}

// Done returns a channel that's closed when streaming is complete
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

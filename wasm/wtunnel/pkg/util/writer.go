package util

import (
	"errors"
	"io"
	"sync"
)

// ErrWriterClosed is returned when writing to a closed AsyncWriter.
var ErrWriterClosed = errors.New("async writer closed")

// WriterFunc allows you to take any function that implements
// the io.Writer interface and turn it into a writer.
type WriterFunc func(p []byte) (n int, err error)

func (f WriterFunc) Write(p []byte) (n int, err error) {
	return f(p)
}

// AsyncWriter wraps a write function with a bounded FIFO queue drained by a
// single worker goroutine. When the queue is full, the oldest queued packet is
// dropped to bound latency and memory. Errors from the underlying writer are
// recorded and propagated to subsequent Write calls.
type AsyncWriter struct {
	fn      func(p []byte) (int, error)
	onError func(error)

	mu     sync.Mutex
	ch     chan []byte
	closed bool
	err    error

	wg sync.WaitGroup
}

// NewAsyncWriter creates a bounded FIFO async writer with capacity max.
// Panics if max <= 0.
func NewAsyncWriter(f func(p []byte) (n int, err error), max int, onError func(error)) *AsyncWriter {
	if max <= 0 {
		panic("max must be greater than zero")
	}
	aw := &AsyncWriter{
		fn:      f,
		onError: onError,
		ch:      make(chan []byte, max),
	}
	aw.wg.Add(1)
	go aw.worker()
	return aw
}

func (aw *AsyncWriter) worker() {
	defer aw.wg.Done()
	for payload := range aw.ch {
		if _, err := aw.fn(payload); err != nil {
			aw.mu.Lock()
			if aw.err == nil {
				aw.err = err
			}
			aw.mu.Unlock()
			if aw.onError != nil {
				aw.onError(err)
			}
		}
	}
}

// Write copies p and enqueues it in FIFO order, dropping the oldest queued
// packet if the buffer is full. Returns any recorded underlying write error
// or ErrWriterClosed if closed.
func (aw *AsyncWriter) Write(p []byte) (int, error) {
	b := make([]byte, len(p))
	copy(b, p)

	aw.mu.Lock()
	defer aw.mu.Unlock()

	if aw.closed {
		return 0, ErrWriterClosed
	}
	if aw.err != nil {
		return 0, aw.err
	}

	select {
	case aw.ch <- b:
	default:
		// Drop the oldest packet to make room for the newest one.
		select {
		case <-aw.ch:
		default:
		}
		aw.ch <- b
	}
	return len(p), nil
}

// Err returns the first error encountered by the underlying writer, if any.
func (aw *AsyncWriter) Err() error {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	return aw.err
}

// Close stops the worker goroutine and waits for queued items to drain.
// Safe to call multiple times.
func (aw *AsyncWriter) Close() error {
	aw.mu.Lock()
	if !aw.closed {
		aw.closed = true
		close(aw.ch)
	}
	aw.mu.Unlock()
	aw.wg.Wait()
	return nil
}

var _ io.WriteCloser = (*AsyncWriter)(nil)

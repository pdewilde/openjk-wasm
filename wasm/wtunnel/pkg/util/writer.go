package util

import (
	"log"
	"sync"
)

// WriterFunc allows you to take any function that implements
// the io.Writer interface and turn it into a writer.
type WriterFunc func(p []byte) (n int, err error)

func (f WriterFunc) Write(p []byte) (n int, err error) {
	return f(p)
}

// AsyncDropOldPacketsWriter wraps a writer function f. A maximum
// of max calls can be buffered up. If a new call comes in after that,
// then the oldest buffered call will be dropped.
// If blockEmpty is set to true, then if there are no current calls blocking
// to the writer AsyncDropOldPacketsWriter will block and return the actual writer
// values.
// AsyncDropOldPacketsWriter always returns success if a packet gets buffered
func AsyncDropOldPacketsWriter(f func(p []byte) (n int, err error), max int32, blockEmpty bool) func(p []byte) (n int, err error) {
	if max < 0 {
		panic("max must be greater than or equal to zero")
	}
	var mut sync.Mutex
	var writerMut sync.Mutex
	queue := make([][]byte, 0, int32(max))
	return func(p []byte) (n int, err error) {
		// If we can just write to the writer without buffering, we will do that
		if blockEmpty && writerMut.TryLock() {
			n, err = f(p)
			writerMut.Unlock()
			return n, err
		}

		mut.Lock()
		defer mut.Unlock()

		b := make([]byte, len(p))
		copy(b, p)

		if int32(len(queue)) >= max {
			log.Println("warning, dropping element from queue") // TODO; remove or set to trace later
			queue = queue[1:]
		}

		queue = append(queue, b)

		// TODO: this is, it should really be a single goroutine rather than one per packet
		// But I need cleanup code. This is self-cleaning!
		go func() {
			writerMut.Lock()
			defer writerMut.Unlock()
			mut.Lock()
			var payload []byte
			if len(queue) > 0 {
				payload = queue[0]
				queue = queue[1:]
			}
			mut.Unlock()
			if payload != nil {
				f(payload)
			}
		}()

		return len(p), nil
	}
}

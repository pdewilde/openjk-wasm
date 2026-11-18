package util

import (
	"bytes"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestAsyncWriterFIFOAndClose(t *testing.T) {
	var mu sync.Mutex
	var got [][]byte

	aw := NewAsyncWriter(func(p []byte) (int, error) {
		mu.Lock()
		got = append(got, append([]byte(nil), p...))
		mu.Unlock()
		return len(p), nil
	}, 8, nil)

	inputs := [][]byte{[]byte("one"), []byte("two"), []byte("three")}
	for _, in := range inputs {
		n, err := aw.Write(in)
		if err != nil || n != len(in) {
			t.Fatalf("Write(%q) = (%d, %v)", in, n, err)
		}
	}
	if err := aw.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	// Idempotent close
	if err := aw.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}

	if _, err := aw.Write([]byte("after")); !errors.Is(err, ErrWriterClosed) {
		t.Fatalf("expected ErrWriterClosed after Close, got %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != len(inputs) {
		t.Fatalf("expected %d packets, got %d", len(inputs), len(got))
	}
	for i := range inputs {
		if !bytes.Equal(got[i], inputs[i]) {
			t.Fatalf("packet %d = %q, want %q", i, got[i], inputs[i])
		}
	}
}

func TestAsyncWriterDropsOldestWhenFull(t *testing.T) {
	gate := make(chan struct{})
	started := make(chan struct{})
	var mu sync.Mutex
	var got []string

	aw := NewAsyncWriter(func(p []byte) (int, error) {
		if string(p) == "first" {
			close(started)
			<-gate
		}
		mu.Lock()
		got = append(got, string(p))
		mu.Unlock()
		return len(p), nil
	}, 2, nil)

	// Block worker on "first"
	if _, err := aw.Write([]byte("first")); err != nil {
		t.Fatal(err)
	}
	<-started

	// Fill capacity=2 with "second", "third", then overflow with "fourth" (should drop "second")
	for _, s := range []string{"second", "third", "fourth"} {
		if _, err := aw.Write([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}

	close(gate)
	if err := aw.Close(); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	want := []string{"first", "third", "fourth"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestAsyncWriterErrorPropagation(t *testing.T) {
	wantErr := errors.New("boom")
	errCh := make(chan error, 1)

	aw := NewAsyncWriter(func(p []byte) (int, error) {
		return 0, wantErr
	}, 4, func(err error) {
		select {
		case errCh <- err:
		default:
		}
	})
	defer aw.Close()

	if _, err := aw.Write([]byte("trigger")); err != nil {
		t.Fatalf("first Write returned unexpected error: %v", err)
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, wantErr) {
			t.Fatalf("onError got %v, want %v", err, wantErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for onError")
	}

	if _, err := aw.Write([]byte("next")); !errors.Is(err, wantErr) {
		t.Fatalf("expected subsequent Write to return %v, got %v", wantErr, err)
	}
	if !errors.Is(aw.Err(), wantErr) {
		t.Fatalf("Err() = %v, want %v", aw.Err(), wantErr)
	}
}

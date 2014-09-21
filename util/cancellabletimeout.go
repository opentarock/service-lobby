package util

import (
	"sync"
	"time"
)

// CancellableTimeout is a utility for timeouts that can be cancelled.
type CancellableTimeout struct {
	cancelled bool
	lock      *sync.Mutex
	cancel    chan<- struct{}
}

// StartCancellableTimeout starts a new timeout with duration and function f
// that is executed if timeout occurs.
func StartCancellableTimeout(duration time.Duration, f func()) *CancellableTimeout {
	cancel := make(chan struct{}, 1)
	go func() {
		select {
		case <-cancel:
			break
		case <-time.After(duration):
			f()
		}
	}()
	return &CancellableTimeout{
		cancel:    cancel,
		cancelled: false,
		lock:      new(sync.Mutex),
	}
}

// Cancel cancels a timeout. In the case of timeout happening concurrently
// with cancelling only one of the two is executed.
func (t *CancellableTimeout) Cancel() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if !t.cancelled {
		t.cancel <- struct{}{}
	}
}

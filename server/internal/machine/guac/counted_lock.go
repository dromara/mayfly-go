package guac

import (
	"sync"
	"sync/atomic"
)

// CountedLock counts how many goroutines are waiting on the lock
type CountedLock struct {
	core     sync.Mutex
	numLocks int32
}

// Lock locks the mutex
func (r *CountedLock) Lock() {
	atomic.AddInt32(&r.numLocks, 1)
	r.core.Lock()
}

// Unlock unlocks the mutex
func (r *CountedLock) Unlock() {
	atomic.AddInt32(&r.numLocks, -1)
	r.core.Unlock()
}

// HasQueued returns true if a goroutine is waiting on the lock
func (r *CountedLock) HasQueued() bool {
	return atomic.LoadInt32(&r.numLocks) > 1
}

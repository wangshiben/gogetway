package lockMap

import (
	"context"
	"sync/atomic"
	"time"
)

// DefaultRWLock it may happen a starve condition
type DefaultRWLock struct {
	Lock
	readCount atomic.Int32
}

func (d *DefaultRWLock) RLock() {
	rwLoc, ok := d.Lock.(*DefaultLock)
	if ok {
		rwLoc.lastCalled = time.Now().UnixMilli()
		rwLoc.lock.RLock()
	}
	if d.readCount.Load() == 0 {
		d.Lock.Lock()
	} else {
		d.readCount.Add(1)
	}

}
func (d *DefaultRWLock) RUnlock() {
	rwLoc, ok := d.Lock.(*DefaultLock)
	if ok {
		rwLoc.lastCalled = time.Now().UnixMilli()
		rwLoc.lock.RUnlock()
	}
	d.readCount.Add(-1)
	if d.readCount.Load() == 0 {
		d.Lock.Unlock()
	}
}

func RWLockDefaultWithCtx(ctx context.Context) RWLock {
	withCtx := LockDefaultWithCtx(ctx)
	return &DefaultRWLock{
		Lock:      withCtx,
		readCount: atomic.Int32{},
	}
}

func RWLockDefaultWithOther(other interface{}) RWLock {
	withOther := LockDefaultWithOther(other)
	return &DefaultRWLock{
		Lock:      withOther,
		readCount: atomic.Int32{},
	}
}

package lockMap

import (
	"context"
	"sync/atomic"
	"time"
)

// DefaultRWLock it may happen a starve condition
type DefaultRWLock struct {
	Locked    Lock
	readCount atomic.Int32
}

func (d *DefaultRWLock) Lock() {
	d.Locked.Lock()
}

func (d *DefaultRWLock) Unlock() {
	d.Locked.Unlock()
}

func (d *DefaultRWLock) Other() interface{} {
	return d.Locked.Other()
}

func (d *DefaultRWLock) UpdateOther(other interface{}) error {
	return d.Locked.UpdateOther(other)
}

func (d *DefaultRWLock) IsLocked() bool {
	return d.Locked.IsLocked()
}

func (d *DefaultRWLock) GetIndex() uint64 {
	return d.Locked.GetIndex()
}

func (d *DefaultRWLock) LastCalled() int64 {
	return d.Locked.LastCalled()
}

func (d *DefaultRWLock) Release(count uint) {
	d.Locked.Release(count)
}

func (d *DefaultRWLock) CanRelease() bool {
	return d.Locked.CanRelease()
}

func (d *DefaultRWLock) IncreaseGetIndex() uint64 {
	return d.Locked.IncreaseGetIndex()
}

func (d *DefaultRWLock) AddRelease() {
	d.Locked.AddRelease()
}

func (d *DefaultRWLock) RLock() {
	rwLoc, ok := d.Locked.(*DefaultLock)
	if ok {
		rwLoc.lastCalled = time.Now().UnixMilli()
		rwLoc.lock.RLock()
	}
	if d.readCount.Load() == 0 {
		d.Locked.Lock()
	} else {
		d.readCount.Add(1)
	}

}
func (d *DefaultRWLock) RUnlock() {
	rwLoc, ok := d.Locked.(*DefaultLock)
	if ok {
		rwLoc.lastCalled = time.Now().UnixMilli()
		rwLoc.lock.RUnlock()
	}
	d.readCount.Add(-1)
	if d.readCount.Load() == 0 {
		d.Locked.Unlock()
	}
}

func RWLockDefaultWithCtx(ctx context.Context) RWLock {
	withCtx := LockDefaultWithCtx(ctx, 0)
	return &DefaultRWLock{
		Locked:    withCtx,
		readCount: atomic.Int32{},
	}
}

func RWLockDefaultWithOther(other interface{}) RWLock {
	withOther := LockDefaultWithOther(other, 0)
	return &DefaultRWLock{
		Locked:    withOther,
		readCount: atomic.Int32{},
	}
}

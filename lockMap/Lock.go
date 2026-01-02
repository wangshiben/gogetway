package lockMap

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type DefaultLock struct {
	lock           sync.RWMutex
	index          uint64
	isLocked       bool
	other          interface{}
	waitingRelease atomic.Int32 // 当前锁占用的等待"释放"的锁数,当<0时，lockGroup可从中直接删除此锁
	lastCalled     int64        //TODO: 死锁自动检测/释放依据
}

func (d *DefaultLock) Lock() {
	d.lock.Lock()
	d.isLocked = true
	d.lastCalled = time.Now().UnixMilli()
}

func (d *DefaultLock) Release(count uint) {
	d.waitingRelease.Add(-int32(count))
}
func (d *DefaultLock) AddRelease() {
	d.waitingRelease.Add(1)
}

func (d *DefaultLock) CanRelease() bool {
	// 没有被锁且没有占用的goruntime可直接释放
	return d.waitingRelease.Load() <= 0 && !d.isLocked
}
func (d *DefaultLock) Unlock() {
	d.lock.Unlock()
	d.isLocked = false
	d.lastCalled = time.Now().UnixMilli()
}

func (d *DefaultLock) Other() interface{} {
	return d.other
}

func (d *DefaultLock) IsLocked() bool {
	return d.isLocked
}

func (d *DefaultLock) GetIndex() uint64 {
	return d.index
}

func (d *DefaultLock) LastCalled() int64 {
	return d.lastCalled
}
func (d *DefaultLock) IncreaseGetIndex() uint64 {
	d.lock.Lock()
	defer d.lock.Unlock()
	index := d.index
	d.index += 1
	return index
}
func LockDefaultWithCtx(ctx context.Context, index uint64) Lock {
	return &DefaultLock{
		lock:           sync.RWMutex{},
		index:          0,
		isLocked:       false,
		other:          ctx.Value("other"),
		waitingRelease: atomic.Int32{},
		lastCalled:     time.Now().UnixMilli(),
	}
}

func LockDefaultWithOther(other interface{}, index uint64) Lock {
	return &DefaultLock{
		lock:           sync.RWMutex{},
		index:          0,
		isLocked:       false,
		other:          other,
		waitingRelease: atomic.Int32{},
		lastCalled:     time.Now().UnixMilli(),
	}
}

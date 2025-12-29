package UsefullStructs

import "sync"

type LockValue[T comparable] struct {
	value    T
	lock     sync.RWMutex
	lockedID T
}
type UnlockFunc func()

func (l *LockValue[T]) Get() T {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.value
}
func (l *LockValue[T]) LockGet() (T, UnlockFunc) {
	l.lock.Lock()
	l.lockedID = l.value
	return l.value, func() {
		l.lock.Unlock()
	}
}
func (l *LockValue[T]) Set(value T) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.value = value
}
func (l *LockValue[T]) SetInLock(value, lockedValue T) {
	if l.lockedID == lockedValue {
		l.value = value
	}
}
func NewLockValue[T comparable](value T) *LockValue[T] {
	return &LockValue[T]{
		value: value,
		lock:  sync.RWMutex{},
	}
}

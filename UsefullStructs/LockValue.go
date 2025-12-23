package UsefullStructs

import "sync"

type LockValue[T any] struct {
	value T
	lock  sync.RWMutex
}

func (l *LockValue[T]) Get() T {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.value
}
func (l *LockValue[T]) Set(value T) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.value = value
}
func NewLockValue[T any](value T) *LockValue[T] {
	return &LockValue[T]{
		value: value,
		lock:  sync.RWMutex{},
	}
}

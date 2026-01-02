package lockMap

import "context"

type LockGroup interface {
	// GetLockedGroup : you can achieve a sub lock group, to avoid increase a lock/lock group block a hole lock get
	// GetLockedGroup : 获取一个锁组，避免频繁创建锁导致整个locks被阻塞
	GetLockedGroup(ctx context.Context, From string) (nextGroup LockGroup, isContinue bool, err error)
	// GetLock : get a lock, when we get a new LockGroup ,we try to call GetLock to get a lock,if not ,we will call GetLockGroup to get next lock group
	// GetLock : 获取一个锁，当获取到一个新的锁组时，我们尝试调用GetLock获取一个锁，如果没有，则调用GetLockGroup获取下一个锁组
	GetLock(ctx context.Context, From string) (lock Lock, err error)
	// NewLockOrGroup : create a lock or lock group,we suggest that you can create a lock or lock group,just one is ok
	// NewLockOrGroup : 创建一个锁或者锁组，我们建议你只创建一个锁或者锁组就够了
	NewLockOrGroup(ctx context.Context, From string) (lock Lock, lockGroup LockGroup, err error)
	// CreateLockGroup : create a lock group , when GetLockGroup return isContinue is false and GetLock returns nil Lock , you have to decide call CreateLockGroup or just CreatLock
	// CreateLockGroup : 创建一个锁组，当GetLockGroup返回isContinue为false且GetLock返回nil Lock时，你必须决定调用CreateLockGroup或者CreateLock
	CreateLockGroup(ctx context.Context, From string) (lockGroup LockGroup, err error)
	CreateLock(ctx context.Context, From string) (lock Lock, err error)
	// Destroy : destroy a lock group when a hook CheckLocks find a lock group is not used,but we suggest you must be careful when you call Destroy
	// Destroy : 销毁一个锁组，当调用CheckLocks检查一个锁组没有被使用时就会调用，我们建议你务必小心调用Destroy
	Destroy()
	CanDestroy() bool

	// DefaultLock : get a default lock, when you want to call NewLockOrGroup
	// DefaultLock : 获取一个默认锁，当你需要调用NewLockOrGroup时
	DefaultLock() RWLock
	// FilterChains : when you decided to start filter packets , we will call FilterChains when we arrive a new  LockGroup
	// FilterChains : 当你决定开始过滤数据包时，我们会在到达一个新的锁组时调用FilterChains
	FilterChains() []FilterChain
	// CheckLocks : it will be called in the case as follows: 1. when we have no any memory 2. every 1min or 10min(that belong your settings)
	// CheckLocks : 当内存不足时，或者每1min或者10min（根据你的设置）时都会调用
	CheckLocks(ctx context.Context)
}
type Lock interface {
	Lock()
	Unlock()
	Other() interface{}
	IsLocked() bool
	GetIndex() uint64
	LastCalled() int64
	Release(count uint)
	CanRelease() bool
	IncreaseGetIndex() uint64
	AddRelease()
}
type RWLock interface {
	Lock
	RLock()
	RUnlock()
}
type CtxWithValue interface {
	context.Context
	Put(key string, value any)
	Clear()
}

type FilterChain func(ctx context.Context, From string, bytes []byte) (isContinue bool, err error)

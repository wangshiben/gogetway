package lockMap

import "context"

type LockGroup interface {
	GetLockedGroup(ctx context.Context, From string) (nextGroup LockGroup, isContinue bool, err error)
	GetLock(ctx context.Context, From string) (lock Lock, err error)
	Destroy()
	DefaultLock() Lock
	FilterChains() []FilterChain
}
type Lock interface {
	Lock()
	Unlock()
	GetIndex() uint64
}

type FilterChain func(ctx context.Context, From string, bytes []byte) (isContinue bool, err error)

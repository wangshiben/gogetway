package lockMap

import (
	"context"
	"errors"
	"gogetway/UsefullStructs"
	"strings"
	"time"
)

type DefaultLockGroup struct {
	lockGroupMap *UsefullStructs.SafeMap[LockGroup]
	lockMap      *UsefullStructs.SafeMap[Lock]
	defaultLock  RWLock
	maxDeep      uint64
	destroyed    bool
}

func (d *DefaultLockGroup) getDepth(ctx context.Context) (uint64, error) {
	deeps, ok := ctx.Value("deeps").(uint64)
	if !ok {
		deeps = 0
	}
	if deeps > d.maxDeep {
		return 0, errors.New("too deep , maybe I'm in a infinite loop")
	}
	return deeps, nil
}
func (d *DefaultLockGroup) deepCheck(ctx context.Context) error {
	_, err := d.getDepth(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (d *DefaultLockGroup) GetLockedGroup(ctx context.Context, From string) (nextGroup LockGroup, isContinue bool, err error) {
	Froms := strings.SplitN(From, ".", 2)
	deeps, err := d.getDepth(ctx)
	if err != nil {
		return nil, false, err
	}
	ctxVal, ok := ctx.(CtxWithValue)
	if ok {
		ctxVal.Put("deeps", uint64(deeps+1))
	} else {
		ctx = context.WithValue(ctx, "deeps", uint64(deeps+1))
	}

	if len(Froms) >= 2 {
		d.defaultLock.RLock()
		defer d.defaultLock.RUnlock()
		value, o := d.lockGroupMap.Get(Froms[0])
		if o {
			return value.GetLockedGroup(ctx, Froms[1])
		}
	} else if len(Froms) == 1 {
		strings.SplitN(Froms[0], ":", 2)
		return d, false, nil
	}
	_, group, err := d.NewLockOrGroup(ctx, From)
	if err != nil {
		return nil, false, err
	}
	if group == nil { // create a new lock
		return d, false, nil
	}
	return group, false, nil
}

func (d *DefaultLockGroup) GetLock(ctx context.Context, From string) (lock Lock, err error) {
	// try to find out in my  lock
	if d.deepCheck(ctx) != nil {
		return nil, err
	}
	Froms := strings.SplitN(From, ":", 2)
	if len(Froms) != 2 {
		return nil, errors.New("lock name error,you may have to check address")
	}
	value, ok := d.lockMap.Get(Froms[1])
	if ok {
		return value, nil
	}
	lock, group, err := d.NewLockOrGroup(ctx, From)
	if err != nil {
		return nil, err
	}
	if group != nil {
		// this will continue until create a new lock,so you don't mind it returns the lock group
		return group.GetLock(ctx, From)
	}
	return lock, nil
}

func (d *DefaultLockGroup) NewLockOrGroup(ctx context.Context, From string) (lock Lock, lockGroup LockGroup, err error) {
	depth, err := d.getDepth(ctx)
	if err != nil {
		return nil, nil, err
	}
	if depth+1 > d.maxDeep {
		return nil, nil, errors.New("too deep , maybe I'm in a infinite loop")
	}
	Froms := strings.SplitN(From, ".", 2)
	if len(Froms) >= 2 {
		group, err := d.CreateLockGroup(ctx, Froms[0])
		if err != nil {
			return nil, group, err
		}
	}
	// create a new lock/ or need to create the last sub lockGroup
	ports := strings.SplitN(Froms[0], ":", 2)
	if len(ports) >= 2 {
		group, err := d.CreateLockGroup(ctx, ports[0])
		if err != nil {
			return nil, group, err
		}
	}
	resLock, err := d.CreateLock(ctx, ports[0])
	return resLock, nil, err
}

func (d *DefaultLockGroup) CreateLockGroup(ctx context.Context, From string) (lockGroup LockGroup, err error) {
	depth, err := d.getDepth(ctx)
	if err != nil {
		return nil, err
	}
	if depth+1 > d.maxDeep {
		return nil, errors.New("too deep , maybe I'm in a infinite loop")
	}
	subGroup := &DefaultLockGroup{
		lockGroupMap: UsefullStructs.NewSafeMap[LockGroup](),
		lockMap:      UsefullStructs.NewSafeMap[Lock](),
		defaultLock:  RWLockDefaultWithCtx(ctx),
		maxDeep:      d.maxDeep,
	}
	d.lockGroupMap.Set(From, subGroup)
	return subGroup, nil
}

func (d *DefaultLockGroup) CreateLock(ctx context.Context, From string) (lock Lock, err error) {
	err = d.deepCheck(ctx)
	if err != nil {
		return nil, err
	}
	sublock := LockDefaultWithCtx(ctx, 0)
	d.lockMap.Set(From, sublock)
	return sublock, nil
}

func (d *DefaultLockGroup) Destroy() {
	d.lockMap = nil
	d.lockGroupMap = nil
	d.destroyed = true
}

func (d *DefaultLockGroup) DefaultLock() RWLock {
	return d.defaultLock
}

func (d *DefaultLockGroup) FilterChains() []FilterChain {
	//TODO implement me
	return nil
}
func (d *DefaultLockGroup) CanDestroy() bool {
	return d.destroyed
}

func (d *DefaultLockGroup) CheckLocks(ctx context.Context) {
	//TODO implement me
	startTime, ok := ctx.Value("started").(int64)
	if !ok {
		startTime = time.Now().UnixMilli()
		ctx = context.WithValue(ctx, "started", startTime)
	}
	d.lockGroupMap.Iterator(func(key string, value LockGroup) {
		if value.CanDestroy() {
			d.lockGroupMap.Delete(key)
		}
		value.CheckLocks(ctx)
	})
	d.lockMap.Iterator(func(key string, value Lock) {
		if value.CanRelease() {
			d.lockMap.Delete(key)
		}
		if value.IsLocked() && time.Now().UnixMilli()-value.LastCalled() > 1000 {
			value.Unlock()
		}
	})
}

func NewDefaultLockGroup(maxDeep uint64) *DefaultLockGroup {
	return &DefaultLockGroup{
		lockGroupMap: UsefullStructs.NewSafeMap[LockGroup](),
		lockMap:      UsefullStructs.NewSafeMap[Lock](),
		defaultLock:  RWLockDefaultWithCtx(context.Background()),
		maxDeep:      maxDeep,
	}
}

package getwayServer

import (
	"context"
	"errors"
	"gogetway/lockMap"
	"net"
	"os"
)

type DefaultResourceGroup struct {
	rootLockGroup lockMap.LockGroup
	defaultWriter *os.File
	writeFunc     WriteFunc
}

func (d *DefaultResourceGroup) GetResource(ctx context.Context, Connect net.Conn) (resource ConnectResource, err error) {
	//TODO implement me
	addr := Connect.RemoteAddr().String()
	lock, err := d.rootLockGroup.GetLock(ctx, addr)
	if err != nil {
		return nil, err
	}
	connectResource, ok := lock.Other().(ConnectResource)
	if !ok {
		if connectResource != nil {
			return nil, errors.New("lock other is not a ConnectResource")
		}
		value := context.WithValue(ctx, "lock", lock)
		resourceFunc := d.NewResourceFunc(value, addr)
		lock.IncreaseGetIndex()
		connectResource, err = resourceFunc(value, addr)
		if err != nil {
			return nil, err
		}
		err = lock.UpdateOther(resource)
		if err != nil {
			return nil, err
		}
	}
	return connectResource, nil
}

func (d *DefaultResourceGroup) NewResourceFunc(ctx context.Context, From string) NewResourceFunc {
	lock, ok := ctx.Value("lock").(lockMap.Lock)
	if !ok {
		return func(ctx context.Context, From string) (resource ConnectResource, err error) {
			return nil, errors.New("pls imp NewResourceFunc")
		}
	}
	return func(ctx context.Context, From string) (resource ConnectResource, err error) {

		return &DefaultResource{
			writer:     d.defaultWriter,
			writeFunc:  nil,
			writeQueue: NewWriteQueue(ctx),
			lock:       lock,
			writeType:  "",
		}, nil
	}
}
func NewResourceGroup(file *os.File, lockGroup lockMap.LockGroup, writeFunc WriteFunc) ResourceGroup {
	return &DefaultResourceGroup{
		rootLockGroup: lockGroup,
		defaultWriter: file,
		writeFunc:     writeFunc,
	}
}

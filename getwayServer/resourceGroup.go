package getwayServer

import (
	"context"
	"errors"
	"gogetway/lockMap"
	"gogetway/writer"
	"io"
	"net"
	"time"
)

type DefaultResourceGroup struct {
	rootLockGroup lockMap.LockGroup
	defaultWriter io.Writer
	writeFunc     WriteFunc
}

func (d *DefaultResourceGroup) GetResource(ctx context.Context, Connect net.Conn) (resource ConnectResource, CloseHook ConnectionCloseHook, err error) {
	//TODO implement me
	addr := Connect.RemoteAddr().String()
	lock, err := d.rootLockGroup.GetLock(ctx, addr)
	if err != nil {
		return nil, nil, err
	}
	lock.AddRelease()
	connectResource, ok := lock.Other().(ConnectResource)
	if !ok {
		if connectResource != nil {
			return nil, nil, errors.New("lock other is not a ConnectResource")
		}
		value := context.WithValue(ctx, "lock", lock)
		resourceFunc := d.NewResourceFunc(value, addr)
		lock.IncreaseGetIndex()
		connectResource, err = resourceFunc(value, addr)
		if err != nil {
			return nil, nil, err
		}
		err = lock.UpdateOther(resource)
		if err != nil {
			return nil, nil, err
		}
	}
	return connectResource, func(resource ConnectResource) {
		lock.Release(1)
	}, nil
}

func (d *DefaultResourceGroup) startCheckingResource() {
	for {
		time.Sleep(10 * time.Minute)
		ctx := context.Background()
		d.rootLockGroup.CheckLocks(ctx)
	}
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
func (d *DefaultResourceGroup) Close() error {
	closer, ok := d.defaultWriter.(io.Closer)
	if ok {
		return closer.Close()
	}
	flushable, ok := d.defaultWriter.(writer.Flushable)
	if ok {
		return flushable.Flush()
	}
	return nil
}
func NewResourceGroup(file io.Writer, lockGroup lockMap.LockGroup, writeFunc WriteFunc) ResourceGroup {
	d := &DefaultResourceGroup{
		rootLockGroup: lockGroup,
		defaultWriter: file,
		writeFunc:     writeFunc,
	}
	go d.startCheckingResource()
	return d
}

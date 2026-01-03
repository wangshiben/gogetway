package reader

import (
	"errors"
	"gogetway/UsefullStructs"
	"gogetway/lockMap"
	"io"
	"sync"
)

type NetTeeReader struct {
	teeReader io.Reader
	lockIndex *UsefullStructs.LockValue[uint64]
	lock      lockMap.Lock
	isRead    bool
	loadIndex uint64
	readLock  sync.RWMutex
	readHook  ReadHook
}
type ReadHook func(index uint64) (isContinue bool, err error)

func (n *NetTeeReader) Read(bytes []byte) (int, error) {
	//fmt.Println("readCalled")
	firstBytes := make([]byte, 1)
	hook := false
	isRead, err := n.teeReader.Read(firstBytes)
	if isRead > 0 {
		get := n.lock.IncreaseGetIndex()
		index := get
		n.loadIndex = index
		n.isRead = true
		bytes[0] = firstBytes[0]
		hook = true
	}
	read, err := n.teeReader.Read(bytes[1:])
	if err != nil {
		return isRead, err
	}
	if hook {
		readHook, err := n.readHook(n.loadIndex)
		if err != nil {
			return 0, err
		}
		if !readHook {
			return 0, errors.New("filter req")
		}
	}

	return read + isRead, err
}
func (n *NetTeeReader) GetIndexed() (uint64, error) {
	n.readLock.RLock()
	defer n.readLock.RUnlock()
	if !n.isRead {
		return 0, errors.New("not read")
	}
	return n.loadIndex, nil
}
func NewNetTeeReader(teeReader io.Reader, LockedValue lockMap.Lock, readHook ReadHook) *NetTeeReader {
	return &NetTeeReader{
		teeReader: teeReader,
		//lockIndex: LockedValue,
		readLock: sync.RWMutex{},
		lock:     LockedValue, // TODO: replace with lockMap.Lock
		readHook: readHook,
	}
}

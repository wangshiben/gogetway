package proto

import "github.com/wangshiben/gogetway/Types"

type Pack interface {
	Marshal() []byte
	Timestamp() int64
	Type() Types.ClientType
	Data() []byte
	From() string
	To() string
}

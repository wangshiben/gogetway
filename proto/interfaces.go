package proto

import "gogetway/Types"

type Pack interface {
	Marshal() []byte
	Timestamp() int64
	Type() Types.ClientType
	Data() []byte
	From() string
	To() string
}

package keys

type Key interface {
	PrimaryKey() []byte
	SecondaryKey() []byte
	FromPrimaryKeyBytes(b []byte) Key
}

type String string

type Uint8 uint8

type Uint32 uint32

type Uint64 uint64

type Int64 int64

package keys

type Key interface {
	PrimaryKey() []byte
	SecondaryKey() []byte
	FromPrimaryKeyBytes(b []byte) Key
}

type String string

func (s String) PrimaryKey() []byte {
	return []byte(s)
}
func (s String) SecondaryKey() []byte {
	return []byte(s)
}

func (s String) FromPrimaryKeyBytes(b []byte) Key {
	return String(b)
}

type Uint8 uint8

type Uint32 uint32

type Uint64 uint64

type Int64 int64

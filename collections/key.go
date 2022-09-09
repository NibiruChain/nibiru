package collections

type Key interface {
	PrimaryKey() []byte
	SecondaryKey() []byte
	FromPrimaryKeyBytes(b []byte) Key
}

type StringKey string

func (s StringKey) PrimaryKey() []byte { return []byte(s) }

func (s StringKey) SecondaryKey() []byte { return append([]byte(s), 0x00) }

func (s StringKey) FromPrimaryKeyBytes(b []byte) Key {
	return StringKey(b)
}

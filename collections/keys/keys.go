package keys

import "fmt"

type Key interface {
	PrimaryKey
	// Stringer is implemented to allow human-readable formats, especially important in errors.
	fmt.Stringer
}

type PrimaryKey interface {
	PrimaryKey() []byte
	FromPrimaryKeyBytes(b []byte) Key
}

// String converts any member of the string typeset into a StringKey
// NOTE(mercilex): this exists to avoid type errors in which bytes are being
// converted to a StringKey which is not correct behavior.
func String[T ~string](v T) StringKey {
	return StringKey(v)
}

type StringKey string

func (s StringKey) PrimaryKey() []byte {
	return []byte(s)
}

func (s StringKey) FromPrimaryKeyBytes(b []byte) Key {
	return StringKey(b)
}

func (s StringKey) String() string {
	return string(s)
}

type Uint8 uint8

type Uint32 uint32

type Uint64 uint64

type Int64 int64

type Order uint8

const (
	OrderAscending Order = iota
	OrderDescending
)

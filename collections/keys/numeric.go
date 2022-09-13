package keys

import "fmt"

type Uint8Key uint8

func (u Uint8Key) KeyBytes() []byte {
	return []byte{uint8(u)}
}

func (u Uint8Key) FromKeyBytes(b []byte) (i int, k Key) {
	return 0, Uint8Key(b[0])
}

func (u Uint8Key) String() string { return fmt.Sprintf("%d", u) }

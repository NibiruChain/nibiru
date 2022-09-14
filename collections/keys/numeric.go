package keys

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Uint8Key uint8

func (u Uint8Key) KeyBytes() []byte {
	return []byte{uint8(u)}
}

func (u Uint8Key) FromKeyBytes(b []byte) (i int, k Key) {
	return 1, Uint8Key(b[0])
}

func (u Uint8Key) String() string { return fmt.Sprintf("%d", u) }

func (u Uint8Key) Marshal() ([]byte, error) {
	return []byte{uint8(u)}, nil
}

func (u *Uint8Key) Unmarshal(b []byte) error {
	if len(b) != 1 {
		return fmt.Errorf("invalid bytes type for Uint8Key")
	}
	*u = Uint8Key(b[0])
	return nil
}

func Uint64[T ~uint64](u T) Uint64Key {
	return Uint64Key(u)
}

type Uint64Key uint64

func (u Uint64Key) KeyBytes() []byte {
	return sdk.Uint64ToBigEndian(uint64(u))
}

func (u Uint64Key) FromKeyBytes(b []byte) (i int, k Key) {
	return 8, Uint64(binary.BigEndian.Uint64(b))
}

func (u Uint64Key) String() string {
	return fmt.Sprintf("%d", u)
}

func (u Uint64Key) Marshal() ([]byte, error) {
	return sdk.Uint64ToBigEndian(uint64(u)), nil
}

func (u *Uint64Key) Unmarshal(b []byte) error {
	if len(b) != 8 {
		return fmt.Errorf("invalid bytes type for Uint64Key")
	}
	*u = Uint64(binary.BigEndian.Uint64(b))
	return nil
}

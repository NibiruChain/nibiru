package collections

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type keys struct {
	String     KeyEncoder[string]
	AccAddress KeyEncoder[sdk.AccAddress]
	Time       KeyEncoder[time.Time]
	Uint64     KeyEncoder[uint64]
}

// Keys is a helper struct that groups together all available key encoder types.
var Keys = keys{
	String:     stringKey{},
	AccAddress: accAddressKey{},
	Time:       timeKey{},
	Uint64:     uint64Key{},
}

type stringKey struct{}

func (stringKey) Encode(s string) []byte {
	if err := validString(s); err != nil {
		panic(fmt.Errorf("invalid StringKey: %w", err))
	}
	return append([]byte(s), 0) // null terminate it for safe prefixing
}

func (stringKey) Decode(b []byte) (int, string) {
	l := len(b)
	if l < 2 {
		panic("invalid StringKey bytes")
	}
	for i, c := range b {
		if c == 0 {
			return i + 1, string(b[:i])
		}
	}
	panic(fmt.Errorf("string is not null terminated: %s", b))
}

type uint64Key struct{}

func (uint64Key) Stringify(u uint64) string     { return strconv.FormatUint(u, 10) }
func (uint64Key) Encode(u uint64) []byte        { return sdk.Uint64ToBigEndian(u) }
func (uint64Key) Decode(b []byte) (int, uint64) { return 8, sdk.BigEndianToUint64(b) }

type timeKey struct{}

func (timeKey) Stringify(t time.Time) string { return t.String() }
func (timeKey) Encode(t time.Time) []byte    { return sdk.FormatTimeBytes(t) }
func (timeKey) Decode(b []byte) (int, time.Time) {
	t, err := sdk.ParseTimeBytes(b)
	if err != nil {
		panic(err)
	}
	return len(b), t
}

type accAddressKey struct{}

func (accAddressKey) Stringify(addr sdk.AccAddress) string { return addr.String() }
func (accAddressKey) Encode(addr sdk.AccAddress) []byte {
	return Keys.String.Encode(addr.String())
}
func (accAddressKey) Decode(b []byte) (int, sdk.AccAddress) {
	i, s := Keys.String.Decode(b)
	return i, sdk.MustAccAddressFromBech32(s)
}

func (stringKey) Stringify(s string) string {
	return s
}

func validString(s string) error {
	for i, c := range s {
		if c == 0 {
			return fmt.Errorf("invalid null character at index %d: %s", i, s)
		}
	}
	return nil
}

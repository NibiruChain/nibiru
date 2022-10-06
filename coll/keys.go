package coll

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
	"time"
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

func (stringKey) KeyEncode(s string) []byte {
	if err := validString(s); err != nil {
		panic(fmt.Errorf("invalid StringKey: %w", err))
	}
	return append([]byte(s), 0) // null terminate it for safe prefixing
}

func (stringKey) KeyDecode(b []byte) (int, string) {
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

func (uint64Key) Stringify(u uint64) string        { return strconv.FormatUint(u, 10) }
func (uint64Key) KeyEncode(u uint64) []byte        { return sdk.Uint64ToBigEndian(u) }
func (uint64Key) KeyDecode(b []byte) (int, uint64) { return 8, sdk.BigEndianToUint64(b) }

type timeKey struct{}

func (timeKey) Stringify(t time.Time) string { return t.String() }
func (timeKey) KeyEncode(t time.Time) []byte { return Keys.Uint64.KeyEncode(uint64(t.UnixMilli())) }
func (timeKey) KeyDecode(b []byte) (int, time.Time) {
	i, u := Keys.Uint64.KeyDecode(b)
	return i, time.UnixMilli(int64(u))
}

type accAddressKey struct{}

func (accAddressKey) Stringify(addr sdk.AccAddress) string { return addr.String() }
func (accAddressKey) KeyEncode(addr sdk.AccAddress) []byte {
	return Keys.String.KeyEncode(addr.String())
}
func (accAddressKey) KeyDecode(b []byte) (int, sdk.AccAddress) {
	i, s := Keys.String.KeyDecode(b)
	return i, sdk.MustAccAddressFromBech32(s)
}

func (stringKey) Stringify(s string) string {
	return s
}

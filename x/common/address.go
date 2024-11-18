package common

import (
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func AddrsToStrings(addrs ...sdk.AccAddress) []string {
	var addrStrings []string
	for _, addr := range addrs {
		addrStrings = append(addrStrings, addr.String())
	}
	return addrStrings
}

func StringsToAddrs(strs ...string) []sdk.AccAddress {
	var addrs []sdk.AccAddress
	for _, str := range strs {
		addr := sdk.MustAccAddressFromBech32(str)
		addrs = append(addrs, addr)
	}

	return addrs
}

// TODO: (realu) Move to collections library
var StringValueEncoder collections.ValueEncoder[string] = stringValueEncoder{}

type stringValueEncoder struct{}

func (a stringValueEncoder) Encode(value string) []byte    { return []byte(value) }
func (a stringValueEncoder) Decode(b []byte) string        { return string(b) }
func (a stringValueEncoder) Stringify(value string) string { return value }
func (a stringValueEncoder) Name() string                  { return "string" }

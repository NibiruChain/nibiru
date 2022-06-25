package common

import (
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

package nutil

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	TreasuryPoolModuleAccount = "treasury_pool"
	// TO_MICRO: multiplier for converting between units and micro-units.
	TO_MICRO = int64(1_000_000)

	NIBIRU_TEAM = "nibi1l8dxzwz9d4peazcqjclnkj2mhvtj7mpnkqx85mg0ndrlhwrnh7gskkzg0v"
)

var LocalnetValAddr = func() sdk.AccAddress {
	bz, err := sdk.GetFromBech32(
		"nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
		"nibi",
	)
	if err != nil {
		panic(err)
	}
	if err := sdk.VerifyAddressFormat(bz); err != nil {
		panic(err)
	}
	return sdk.AccAddress(bz)
}()

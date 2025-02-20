package common

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TreasuryPoolModuleAccount = "treasury_pool"
	// TO_MICRO: multiplier for converting between units and micro-units.
	TO_MICRO = int64(1_000_000)

	NIBIRU_TEAM = "nibi1l8dxzwz9d4peazcqjclnkj2mhvtj7mpnkqx85mg0ndrlhwrnh7gskkzg0v"
)

// TO_ATTO eth multiplier
var TO_ATTO = big.NewInt(1e18)

func NibiruTeamAddr() sdk.AccAddress {
	return sdk.MustAccAddressFromBech32(NIBIRU_TEAM)
}

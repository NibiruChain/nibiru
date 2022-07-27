package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		VaultBalance:   sdk.NewCoins(),
		PerpEfBalance:  sdk.NewCoins(),
		FeePoolBalance: sdk.NewCoins(),
		PairMetadata: []*PairMetadata{
			{
				Pair:                       common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
			},
			{
				Pair:                       common.PairETHStable,
				CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
			},
		},
		Positions:            []*Position{},
		PrepaidBadDebts:      []*PrepaidBadDebt{},
		WhitelistedAddresses: []string{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

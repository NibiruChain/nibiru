package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

func TestGenesisState_Validate(t *testing.T) {
	defaultFeeRatio := sdk.MustNewDecFromStr("0.002")
	defaultFeeRatioEF := sdk.MustNewDecFromStr("0.5")
	defaultBonusRateRecoll := sdk.MustNewDecFromStr("0.002")
	adjustmentStep := sdk.MustNewDecFromStr("0.0025")
	priceLowerBound := sdk.MustNewDecFromStr("0.9999")
	priceUpperBound := sdk.MustNewDecFromStr("1.0001")

	testCases := []struct {
		description string
		genState    *types.GenesisState
		expectValid bool
	}{
		{
			description: "default is valid",
			genState:    types.DefaultGenesis(),
			expectValid: true,
		},
		{
			description: "valid genesis state",
			genState:    &types.GenesisState{},
			expectValid: true,
		},
		{
			description: "manually set default params",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
			},
			expectValid: true,
		},
		{
			description: "set non-default, valid collRatio at genesis",
			genState: &types.GenesisState{
				Params: types.NewParams(
					sdk.MustNewDecFromStr("0.7"), defaultFeeRatio, defaultFeeRatioEF, defaultBonusRateRecoll, "15 min", adjustmentStep, priceLowerBound, priceUpperBound, true),
			},
			expectValid: true,
		},
		{
			description: "set invalid negative collRatio at genesis",
			genState: &types.GenesisState{
				Params: types.NewParams(
					sdk.MustNewDecFromStr("-0.5"), defaultFeeRatio, defaultFeeRatioEF, defaultBonusRateRecoll, "15 min", adjustmentStep, priceLowerBound, priceUpperBound, true),
			},
			expectValid: false,
		},
		{
			description: "set invalid > max collRatio at genesis",
			genState: &types.GenesisState{
				Params: types.NewParams(
					sdk.MustNewDecFromStr("1.5"), defaultFeeRatio, defaultFeeRatioEF, defaultBonusRateRecoll, "15 min", adjustmentStep, priceLowerBound, priceUpperBound, true),
			},
			expectValid: false,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.description, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.expectValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

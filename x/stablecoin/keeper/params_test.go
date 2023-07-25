package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

func TestGetParams(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
	stableKeeper := &nibiruApp.StablecoinKeeper

	params := types.DefaultParams()

	stableKeeper.SetParams(ctx, params)

	require.EqualValues(t, params, stableKeeper.GetParams(ctx))
}

func TestNewParams_Errors(t *testing.T) {
	tests := []struct {
		name          string
		params        types.Params
		expectedError error
	}{
		{
			"collateral ratio bigger than 1",
			types.NewParams(
				sdk.MustNewDecFromStr("2"),
				sdk.OneDec(),
				sdk.OneDec(),
				sdk.MustNewDecFromStr("0.002"),
				"15 min",
				sdk.MustNewDecFromStr("0.0025"),
				sdk.MustNewDecFromStr("0.9999"),
				sdk.MustNewDecFromStr("1.0001"),
				true,
			),
			fmt.Errorf(
				"collateral ratio is above max value(1e6): %s",
				sdk.MustNewDecFromStr("2").Mul(sdk.NewDec(1*common.TO_MICRO)).TruncateInt()),
		},
		{
			"fee ratio bigger than 1",
			types.NewParams(
				sdk.OneDec(),
				sdk.MustNewDecFromStr("2"),
				sdk.OneDec(),
				sdk.MustNewDecFromStr("0.002"),
				"15 min",
				sdk.MustNewDecFromStr("0.0025"),
				sdk.MustNewDecFromStr("0.9999"),
				sdk.MustNewDecFromStr("1.0001"),
				true,
			),
			fmt.Errorf(
				"fee ratio is above max value(1e6): %s",
				sdk.MustNewDecFromStr("2").Mul(sdk.NewDec(1*common.TO_MICRO)).TruncateInt()),
		},
		{
			"stable EF fee ratio bigger than 1",
			types.NewParams(
				sdk.OneDec(),
				sdk.OneDec(),
				sdk.MustNewDecFromStr("2"),
				sdk.MustNewDecFromStr("0.002"),
				"15 min",
				sdk.MustNewDecFromStr("0.0025"),
				sdk.MustNewDecFromStr("0.9999"),
				sdk.MustNewDecFromStr("1.0001"),
				false,
			),
			fmt.Errorf(
				"stable EF fee ratio is above max value(1e6): %s",
				sdk.MustNewDecFromStr("2").Mul(sdk.NewDec(1*common.TO_MICRO)).TruncateInt()),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			require.EqualError(t, err, tc.expectedError.Error())
		})
	}
}

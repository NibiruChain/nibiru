package keeper_test

import (
	"testing"
	"time"

	"github.com/MatrixDao/matrix/x/common"
	pricefeedTypes "github.com/MatrixDao/matrix/x/pricefeed/types"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/sample"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSetCollRatio_Input(t *testing.T) {

	type TestCase struct {
		name         string
		inCollRatio  sdk.Dec
		expectedPass bool
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			stablecoinKeeper := &matrixApp.StablecoinKeeper

			err := stablecoinKeeper.SetCollRatio(ctx, tc.inCollRatio)
			if tc.expectedPass {
				require.NoError(
					t, err, "Error setting the CollRatio: %d", tc.inCollRatio)
				return
			}
			require.Error(t, err)
		})
	}

	testCases := []TestCase{
		{
			name:         "Upper bound of CollRatio",
			inCollRatio:  sdk.OneDec(),
			expectedPass: true,
		}, {
			name:         "Lower bound of CollRatio",
			inCollRatio:  sdk.ZeroDec(),
			expectedPass: true,
		}, {
			name:         "CollRatio above 100",
			inCollRatio:  sdk.MustNewDecFromStr("1.5"),
			expectedPass: false,
		}, {
			name:         "Negative CollRatio not allowed",
			inCollRatio:  sdk.OneDec().Neg(),
			expectedPass: false,
		},
	}
	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

func TestGetCollRatio_Input(t *testing.T) {

	testName := "GetCollRatio after setting default params returns expected value"
	t.Run(testName, func(t *testing.T) {

		matrixApp, ctx := testutil.NewMatrixApp()
		stablecoinKeeper := &matrixApp.StablecoinKeeper

		stablecoinKeeper.SetParams(ctx, types.DefaultParams())
		expectedCollRatioInt := sdk.NewInt(types.DefaultParams().CollRatio)

		outCollRatio := stablecoinKeeper.GetCollRatio(ctx)
		outCollRatioInt := outCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		require.EqualValues(t, expectedCollRatioInt, outCollRatioInt)
	})

	testName = "Setting to non-default value returns expected value"
	t.Run(testName, func(t *testing.T) {

		matrixApp, ctx := testutil.NewMatrixApp()
		stablecoinKeeper := &matrixApp.StablecoinKeeper

		expectedCollRatio := sdk.MustNewDecFromStr("0.5")
		expectedCollRatioInt := expectedCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		stablecoinKeeper.SetCollRatio(ctx, expectedCollRatio)

		outCollRatio := stablecoinKeeper.GetCollRatio(ctx)
		outCollRatioInt := outCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		require.EqualValues(t, expectedCollRatioInt, outCollRatioInt)
	})

}

// ---------------------------------------------------------------
// NeededCollUSD tests
// ---------------------------------------------------------------

type TestCaseGetNeededCollUSD struct {
	name            string
	protocolColl    sdk.Int
	priceCollStable sdk.Dec
	stableSupply    sdk.Int
	targetCollRatio sdk.Dec
	neededCollUSD   sdk.Dec

	expectedPass bool
}

func TestGetNeededCollUSD_NoError(t *testing.T) {

	executeTest := func(t *testing.T, testCase TestCaseGetNeededCollUSD) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			stablecoinKeeper := &matrixApp.StablecoinKeeper
			stablecoinKeeper.SetCollRatio(ctx, tc.targetCollRatio)
			stablecoinKeeper.IncreaseModuleCollBalance(
				ctx, sdk.NewCoin(common.CollDenom, tc.protocolColl))
			matrixApp.BankKeeper.MintCoins(
				ctx, types.ModuleName,
				sdk.NewCoins(sdk.NewCoin(common.StableDenom, tc.stableSupply)),
			)

			// Set up markets for the pricefeed keeper.
			marketID := common.CollStablePool
			oracle := sample.AccAddress()
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			pricefeedParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: marketID, BaseAsset: common.CollDenom,
						QuoteAsset: common.StableDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
				}}
			matrixApp.PriceKeeper.SetParams(ctx, pricefeedParams)

			// Post prices to each market with the oracle.
			_, err := matrixApp.PriceKeeper.SetPrice(
				ctx, oracle, marketID, tc.priceCollStable, priceExpiry)
			require.NoError(t, err)

			// Update the 'CurrentPrice' posted by the oracles.
			for _, market := range pricefeedParams.Markets {
				err = matrixApp.PriceKeeper.SetCurrentPrices(ctx, market.MarketID)
				require.NoError(t, err, "Error posting price for market: %d", market)
			}

			neededCollUSD, err := stablecoinKeeper.GetNeededCollUSD(ctx)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.neededCollUSD, neededCollUSD)
			} else {
				require.Error(t, err)
			}
		})
	}

	testCases := []TestCaseGetNeededCollUSD{
		{
			name:            "Too much collateral gives correct positive value",
			protocolColl:    sdk.NewInt(500),
			priceCollStable: sdk.OneDec(), // startCollUSD = 500 * 1 -> 500
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6"), // 0.6 * 1000 = 600
			neededCollUSD:   sdk.MustNewDecFromStr("100"), // = 600 - 500
			expectedPass:    true,
		}, {
			name:            "Too much collateral gives correct negative value",
			protocolColl:    sdk.NewInt(600),
			priceCollStable: sdk.OneDec(), // startCollUSD = 600 * 1 = 600
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.5"),  // 0.5 * 1000 = 500
			neededCollUSD:   sdk.MustNewDecFromStr("-100"), // = 500 - 600
			expectedPass:    true,
		},
	}
	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

func TestGetNeededCollUSD_NoPricePosted(t *testing.T) {

	executeTest := func(t *testing.T, testCase TestCaseGetNeededCollUSD) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			stablecoinKeeper := &matrixApp.StablecoinKeeper
			stablecoinKeeper.SetCollRatio(ctx, tc.targetCollRatio)
			stablecoinKeeper.IncreaseModuleCollBalance(
				ctx, sdk.NewCoin(common.CollDenom, tc.protocolColl))
			matrixApp.BankKeeper.MintCoins(
				ctx, types.ModuleName,
				sdk.NewCoins(sdk.NewCoin(common.StableDenom, tc.stableSupply)),
			)

			// Set up markets for the pricefeed keeper.
			marketID := common.CollStablePool
			oracle := sample.AccAddress()
			pricefeedParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: marketID, BaseAsset: common.CollDenom,
						QuoteAsset: common.StableDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
				}}
			matrixApp.PriceKeeper.SetParams(ctx, pricefeedParams)

			neededCollUSD, err := stablecoinKeeper.GetNeededCollUSD(ctx)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.neededCollUSD, neededCollUSD)
			} else {
				require.Error(t, err)
			}
		})
	}

	testCases := []TestCaseGetNeededCollUSD{
		{
			name:            "No price availabale for the collateral",
			protocolColl:    sdk.NewInt(500),
			priceCollStable: sdk.OneDec(), // startCollUSD = 500 * 1 -> 500
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6"), // 0.6 * 1000 = 600
			neededCollUSD:   sdk.MustNewDecFromStr("100"), // = 600 - 500
			expectedPass:    false,
		}}
	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

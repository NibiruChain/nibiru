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

func TestGetNeededCollUSD(t *testing.T) {

	type TestCaseGetNeededCollUSD struct {
		name            string
		protocolColl    sdk.Int
		priceCollStable sdk.Dec
		postedMarketIDs []string
		stableSupply    sdk.Int
		targetCollRatio sdk.Dec
		neededCollUSD   sdk.Dec

		expectedPass bool
	}

	executeTest := func(t *testing.T, testCase TestCaseGetNeededCollUSD) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			stablecoinKeeper := &matrixApp.StablecoinKeeper
			stablecoinKeeper.SetCollRatio(ctx, tc.targetCollRatio)
			matrixApp.BankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(
					sdk.NewCoin(common.CollDenom, tc.protocolColl),
					sdk.NewCoin(common.StableDenom, tc.stableSupply),
				),
			)

			// Set up markets for the pricefeed keeper.
			oracle := sample.AccAddress()
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			pricefeedParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: common.CollStablePool, BaseAsset: common.CollDenom,
						QuoteAsset: common.StableDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
					{MarketID: common.GovCollPool, BaseAsset: common.GovDenom,
						QuoteAsset: common.CollDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
				}}
			matrixApp.PriceKeeper.SetParams(ctx, pricefeedParams)

			// Post prices to each specified market with the oracle.
			prices := map[string]sdk.Dec{
				common.CollStablePool: tc.priceCollStable,
			}
			for _, marketID := range tc.postedMarketIDs {
				_, err := matrixApp.PriceKeeper.SetPrice(
					ctx, oracle, marketID, prices[marketID], priceExpiry)
				require.NoError(t, err)

				// Update the 'CurrentPrice' posted by the oracles.
				err = matrixApp.PriceKeeper.SetCurrentPrices(ctx, marketID)
				require.NoError(t, err, "Error posting price for market: %d", marketID)
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
			name:            "Too little collateral gives correct positive value",
			protocolColl:    sdk.NewInt(500),
			priceCollStable: sdk.OneDec(), // startCollUSD = 500 * 1 -> 500
			postedMarketIDs: []string{common.CollStablePool},
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6"), // 0.6 * 1000 = 600
			neededCollUSD:   sdk.MustNewDecFromStr("100"), // = 600 - 500
			expectedPass:    true,
		}, {
			name:            "Too much collateral gives correct negative value",
			protocolColl:    sdk.NewInt(600),
			priceCollStable: sdk.OneDec(), // startCollUSD = 600 * 1 = 600
			postedMarketIDs: []string{common.CollStablePool},
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.5"),  // 0.5 * 1000 = 500
			neededCollUSD:   sdk.MustNewDecFromStr("-100"), // = 500 - 600
			expectedPass:    true,
		}, {
			name:            "No price availabale for the collateral",
			protocolColl:    sdk.NewInt(500),
			priceCollStable: sdk.OneDec(), // startCollUSD = 500 * 1 -> 500
			postedMarketIDs: []string{},
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6"), // 0.6 * 1000 = 600
			neededCollUSD:   sdk.MustNewDecFromStr("100"), // = 600 - 500
			expectedPass:    false,
		},
	}
	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

func TestGetNeededCollAmount(t *testing.T) {

	type TestCaseGetNeededCollAmount struct {
		name            string
		protocolColl    sdk.Int
		priceCollStable sdk.Dec
		stableSupply    sdk.Int
		targetCollRatio sdk.Dec
		neededCollAmt   sdk.Int
		expectedPass    bool
	}

	testCases := []TestCaseGetNeededCollAmount{
		{
			name:            "under-collateralized; untruncated integer amount",
			protocolColl:    sdk.NewInt(500),
			priceCollStable: sdk.OneDec(), // startCollUSD = 500 * 1 -> 500
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6"), // 0.6 * 1000 = 600
			neededCollAmt:   sdk.NewInt(100),              // = 600 - 500
			expectedPass:    true,
		},
		{
			name:            "under-collateralized; truncated integer amount",
			protocolColl:    sdk.NewInt(500),
			priceCollStable: sdk.OneDec(), // startCollUSD = 500 * 1 -> 500
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6025"), // 0.6025 * 1000 = 602.5
			neededCollAmt:   sdk.NewInt(103),                 //  602.5 - 500 -> 103 required
			expectedPass:    true,
		},
		{
			name:            "under-collateralized; truncated integer amount; non-unit price",
			protocolColl:    sdk.NewInt(500),
			priceCollStable: sdk.MustNewDecFromStr("0.999"), // startCollUSD = 500 * 0.999 -> 499.5
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.603"), // 0.603 * 1000 = 603
			//  603 - 499.5 = 103.5 -> 104 required
			neededCollAmt: sdk.NewInt(104),
			expectedPass:  true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			stablecoinKeeper := &matrixApp.StablecoinKeeper
			stablecoinKeeper.SetCollRatio(ctx, tc.targetCollRatio)
			matrixApp.BankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(
					sdk.NewCoin(common.CollDenom, tc.protocolColl),
					sdk.NewCoin(common.StableDenom, tc.stableSupply),
				),
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

			neededCollAmount, err := stablecoinKeeper.GetNeededCollAmount(ctx)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.neededCollAmt, neededCollAmount)
			} else {
				require.Error(t, err)
			}
		})
	}

	testCases = []TestCaseGetNeededCollAmount{
		{
			name:            "error from price not being posted",
			protocolColl:    sdk.NewInt(500),
			priceCollStable: sdk.OneDec(), // startCollUSD = 500 * 1 -> 500
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6"), // 0.6 * 1000 = 600
			neededCollAmt:   sdk.NewInt(100),              // = 600 - 500
			expectedPass:    false,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			stablecoinKeeper := &matrixApp.StablecoinKeeper
			stablecoinKeeper.SetCollRatio(ctx, tc.targetCollRatio)
			matrixApp.BankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(
					sdk.NewCoin(common.CollDenom, tc.protocolColl),
					sdk.NewCoin(common.StableDenom, tc.stableSupply),
				),
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

			neededCollAmount, err := stablecoinKeeper.GetNeededCollAmount(ctx)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.neededCollAmt, neededCollAmount)
			} else {
				require.Error(t, err)
			}
		})
	}

}

func TestGovAmtFromRecollateralize(t *testing.T) {

	type TestCaseGovAmtFromRecollateralize struct {
		name            string
		protocolColl    sdk.Int
		priceCollStable sdk.Dec
		priceGovColl    sdk.Dec
		stableSupply    sdk.Int
		targetCollRatio sdk.Dec
		postedMarketIDs []string

		govOut       sdk.Int
		expectedPass bool
	}

	executeTest := func(t *testing.T, testCase TestCaseGovAmtFromRecollateralize) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			stablecoinKeeper := &matrixApp.StablecoinKeeper
			stablecoinKeeper.SetCollRatio(ctx, tc.targetCollRatio)
			matrixApp.BankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(
					sdk.NewCoin(common.CollDenom, tc.protocolColl),
					sdk.NewCoin(common.StableDenom, tc.stableSupply),
				),
			)

			// Set up markets for the pricefeed keeper.
			oracle := sample.AccAddress()
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			pricefeedParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: common.CollStablePool, BaseAsset: common.CollDenom,
						QuoteAsset: common.StableDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
					{MarketID: common.GovCollPool, BaseAsset: common.GovDenom,
						QuoteAsset: common.CollDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
				}}
			matrixApp.PriceKeeper.SetParams(ctx, pricefeedParams)

			// Post prices to each specified market with the oracle.
			prices := map[string]sdk.Dec{
				common.CollStablePool: tc.priceCollStable,
				common.GovCollPool:    tc.priceGovColl,
			}
			for _, marketID := range tc.postedMarketIDs {
				_, err := matrixApp.PriceKeeper.SetPrice(
					ctx, oracle, marketID, prices[marketID], priceExpiry)
				require.NoError(t, err)

				// Update the 'CurrentPrice' posted by the oracles.
				err = matrixApp.PriceKeeper.SetCurrentPrices(ctx, marketID)
				require.NoError(t, err, "Error posting price for market: %d", marketID)
			}

			govOut, err := stablecoinKeeper.GovAmtFromRecollateralize(ctx)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.govOut, govOut)
			} else {
				require.Error(t, err)
			}
		})
	}

	testCases := []TestCaseGovAmtFromRecollateralize{
		{
			name:            "no prices posted",
			protocolColl:    sdk.NewInt(500),
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6"),
			postedMarketIDs: []string{},
			govOut:          sdk.Int{},
			expectedPass:    false,
		},
		{
			name:            "only post collateral price",
			protocolColl:    sdk.NewInt(500),
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6"), // 0.6 * 1000 = 600
			priceCollStable: sdk.OneDec(),
			postedMarketIDs: []string{common.CollStablePool},
			govOut:          sdk.Int{},
			expectedPass:    false,
		},
		{
			name:            "only post gov price",
			protocolColl:    sdk.NewInt(500),
			stableSupply:    sdk.NewInt(1000),
			targetCollRatio: sdk.MustNewDecFromStr("0.6"), // 0.6 * 1000 = 600
			priceGovColl:    sdk.OneDec(),
			postedMarketIDs: []string{common.GovCollPool},
			govOut:          sdk.Int{},
			expectedPass:    false,
		},
		{
			name:            "correct computation - positive",
			protocolColl:    sdk.NewInt(5_000),
			stableSupply:    sdk.NewInt(10_000),
			targetCollRatio: sdk.MustNewDecFromStr("0.7"), // 0.7 * 10_000 = 7_000
			postedMarketIDs: []string{common.GovCollPool, common.CollStablePool},
			priceCollStable: sdk.OneDec(),
			priceGovColl:    sdk.NewDec(2),
			// govOut = neededCollUSD * (1 + bonusRate) / priceGov
			//        = 2000 * (1.002) / 2 = 1002
			govOut:       sdk.NewInt(1002),
			expectedPass: true,
		},
		{
			name:            "correct computation - positive, new price",
			protocolColl:    sdk.NewInt(50_000),
			stableSupply:    sdk.NewInt(100_000),
			targetCollRatio: sdk.MustNewDecFromStr("0.7"), // 0.7 * 100_000 = 70_000
			postedMarketIDs: []string{common.GovCollPool, common.CollStablePool},
			priceCollStable: sdk.OneDec(),
			priceGovColl:    sdk.NewDec(10),
			// govOut = neededCollUSD * (1 + bonusRate) / priceGov
			//        = 20000 * (1.002) / 10 = 2004
			govOut:       sdk.NewInt(2004),
			expectedPass: true,
		},
		{
			name:            "correct computation - negative",
			protocolColl:    sdk.NewInt(70_000),
			stableSupply:    sdk.NewInt(100_000),
			targetCollRatio: sdk.MustNewDecFromStr("0.5"), // 0.5 * 100_000 = 50_000
			postedMarketIDs: []string{common.GovCollPool, common.CollStablePool},
			priceCollStable: sdk.OneDec(),
			priceGovColl:    sdk.NewDec(10),
			// govOut = neededCollUSD * (1 + bonusRate) / priceGov
			//        = -20000 * (1.002) / 10 = 2004
			govOut:       sdk.NewInt(-2004),
			expectedPass: true,
		},
	}

	for _, testCase := range testCases {
		executeTest(t, testCase)
	}

}

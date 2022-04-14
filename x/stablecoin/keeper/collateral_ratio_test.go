package keeper_test

import (
	"testing"
	"time"

	"github.com/MatrixDao/matrix/x/common"
	pricefeedTypes "github.com/MatrixDao/matrix/x/pricefeed/types"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/sample"

	"github.com/cosmos/cosmos-sdk/simapp"
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

			matrixApp, ctx := testutil.NewMatrixApp(true)
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

		matrixApp, ctx := testutil.NewMatrixApp(true)
		stablecoinKeeper := &matrixApp.StablecoinKeeper

		stablecoinKeeper.SetParams(ctx, types.DefaultParams())
		expectedCollRatioInt := sdk.NewInt(types.DefaultParams().CollRatio)

		outCollRatio := stablecoinKeeper.GetCollRatio(ctx)
		outCollRatioInt := outCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		require.EqualValues(t, expectedCollRatioInt, outCollRatioInt)
	})

	testName = "Setting to non-default value returns expected value"
	t.Run(testName, func(t *testing.T) {

		matrixApp, ctx := testutil.NewMatrixApp(true)
		stablecoinKeeper := &matrixApp.StablecoinKeeper

		expectedCollRatio := sdk.MustNewDecFromStr("0.5")
		expectedCollRatioInt := expectedCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		stablecoinKeeper.SetCollRatio(ctx, expectedCollRatio)

		outCollRatio := stablecoinKeeper.GetCollRatio(ctx)
		outCollRatioInt := outCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		require.EqualValues(t, expectedCollRatioInt, outCollRatioInt)
	})

}

func TestGetCollUSDForTargetCollRatio(t *testing.T) {

	type TestCaseGetCollUSDForTargetCollRatio struct {
		name            string
		protocolColl    sdk.Int
		priceCollStable sdk.Dec
		postedMarketIDs []string
		stableSupply    sdk.Int
		targetCollRatio sdk.Dec
		neededCollUSD   sdk.Dec

		expectedPass bool
	}

	executeTest := func(t *testing.T, testCase TestCaseGetCollUSDForTargetCollRatio) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp(true)
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
					{MarketID: common.GovStablePool, BaseAsset: common.GovDenom,
						QuoteAsset: common.StableDenom,
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

			neededCollUSD, err := stablecoinKeeper.GetCollUSDForTargetCollRatio(ctx)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.neededCollUSD, neededCollUSD)
			} else {
				require.Error(t, err)
			}
		})
	}

	testCases := []TestCaseGetCollUSDForTargetCollRatio{
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

func TestGetCollAmtForTargetCollRatio(t *testing.T) {

	type TestCaseGetCollAmtForTargetCollRatio struct {
		name            string
		protocolColl    sdk.Int
		priceCollStable sdk.Dec
		stableSupply    sdk.Int
		targetCollRatio sdk.Dec
		neededCollAmt   sdk.Int
		expectedPass    bool
	}

	testCases := []TestCaseGetCollAmtForTargetCollRatio{
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

			matrixApp, ctx := testutil.NewMatrixApp(true)
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

			neededCollAmount, err := stablecoinKeeper.GetCollAmtForTargetCollRatio(ctx)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.neededCollAmt, neededCollAmount)
			} else {
				require.Error(t, err)
			}
		})
	}

	testCases = []TestCaseGetCollAmtForTargetCollRatio{
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

			matrixApp, ctx := testutil.NewMatrixApp(true)
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

			neededCollAmount, err := stablecoinKeeper.GetCollAmtForTargetCollRatio(ctx)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.neededCollAmt, neededCollAmount)
			} else {
				require.Error(t, err)
			}
		})
	}

}

func TestGovAmtFromFullRecollateralize(t *testing.T) {

	testCases := []struct {
		name            string
		protocolColl    sdk.Int
		priceCollStable sdk.Dec
		priceGovStable  sdk.Dec
		stableSupply    sdk.Int
		targetCollRatio sdk.Dec
		postedMarketIDs []string

		govOut       sdk.Int
		expectedPass bool
	}{
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
			priceGovStable:  sdk.OneDec(),
			postedMarketIDs: []string{common.GovStablePool},
			govOut:          sdk.Int{},
			expectedPass:    false,
		},
		{
			name:            "correct computation - positive",
			protocolColl:    sdk.NewInt(5_000),
			stableSupply:    sdk.NewInt(10_000),
			targetCollRatio: sdk.MustNewDecFromStr("0.7"), // 0.7 * 10_000 = 7_000
			postedMarketIDs: []string{common.GovStablePool, common.CollStablePool},
			priceCollStable: sdk.OneDec(),
			priceGovStable:  sdk.NewDec(2),
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
			postedMarketIDs: []string{common.GovStablePool, common.CollStablePool},
			priceCollStable: sdk.OneDec(),
			priceGovStable:  sdk.NewDec(10),
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
			postedMarketIDs: []string{common.GovStablePool, common.CollStablePool},
			priceCollStable: sdk.OneDec(),
			priceGovStable:  sdk.NewDec(10),
			// govOut = neededCollUSD * (1 + bonusRate) / priceGov
			//        = -20000 * (1.002) / 10 = 2004
			govOut:       sdk.NewInt(-2004),
			expectedPass: true,
		},
	}

	for _, testCase := range testCases {

		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp(true)
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
					{MarketID: common.GovStablePool, BaseAsset: common.GovDenom,
						QuoteAsset: common.StableDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
				}}
			matrixApp.PriceKeeper.SetParams(ctx, pricefeedParams)

			// Post prices to each specified market with the oracle.
			prices := map[string]sdk.Dec{
				common.CollStablePool: tc.priceCollStable,
				common.GovStablePool:  tc.priceGovStable,
			}
			for _, marketID := range tc.postedMarketIDs {
				_, err := matrixApp.PriceKeeper.SetPrice(
					ctx, oracle, marketID, prices[marketID], priceExpiry)
				require.NoError(t, err)

				// Update the 'CurrentPrice' posted by the oracles.
				err = matrixApp.PriceKeeper.SetCurrentPrices(ctx, marketID)
				require.NoError(t, err, "Error posting price for market: %d", marketID)
			}

			govOut, err := stablecoinKeeper.GovAmtFromFullRecollateralize(ctx)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.govOut, govOut)
			} else {
				require.Error(t, err)
			}
		})
	}

}

type NeededCollScenario struct {
	protocolColl    sdk.Int
	priceCollStable sdk.Dec
	stableSupply    sdk.Int
	collRatio       sdk.Dec
}

func (scenario NeededCollScenario) CalcNeededUSD() (neededUSD sdk.Dec) {
	stableUSD := scenario.collRatio.MulInt(scenario.stableSupply)
	collUSD := scenario.priceCollStable.MulInt(scenario.protocolColl)
	return stableUSD.Sub(collUSD)
}

func TestRecollateralize(t *testing.T) {

	testCases := []struct {
		name         string
		expectedPass bool

		postedMarketIDs   []string
		scenario          NeededCollScenario
		priceGovStable    sdk.Dec
		expectedNeededUSD sdk.Dec
		accFunds          sdk.Coins

		msg      types.MsgRecollateralize
		response *types.MsgRecollateralizeResponse
	}{
		{
			name:            "both prices are $1",
			postedMarketIDs: []string{common.CollStablePool, common.GovStablePool},
			scenario: NeededCollScenario{
				protocolColl:    sdk.NewInt(500_000),
				priceCollStable: sdk.OneDec(),
				stableSupply:    sdk.NewInt(1_000_000),
				collRatio:       sdk.MustNewDecFromStr("0.6"),
				// neededCollUSD =  (0.6 * 1000e3) - (500e3 *1) = 100_000
			},
			priceGovStable: sdk.OneDec(),
			accFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.CollDenom, 1_000_000_000),
			),

			expectedNeededUSD: sdk.NewDec(100_000),
			msg: types.MsgRecollateralize{
				Creator: sample.AccAddress().String(),
				Coll:    sdk.NewCoin(common.CollDenom, sdk.NewInt(100_000)),
			},
			response: &types.MsgRecollateralizeResponse{
				/*
					Gov.Amount = inCollUSD * (1 + bonusRate) / priceGovStable
					  = 100_000 * (1.002) / priceGovStable
					  = 100_200 / priceGovStable
				*/
				Gov: sdk.NewCoin(common.GovDenom, sdk.NewInt(100_200)),
			},
			expectedPass: true,
		},
		{
			name:            "arbitrary valid prices",
			postedMarketIDs: []string{common.CollStablePool, common.GovStablePool},
			scenario: NeededCollScenario{
				protocolColl:    sdk.NewInt(500_000),
				priceCollStable: sdk.MustNewDecFromStr("1.099999"),
				stableSupply:    sdk.NewInt(1_000_000),
				collRatio:       sdk.MustNewDecFromStr("0.7"),
				// neededCollUSD =  (0.7 * 1000e3) - (500e3 *1.09999) = 150_000.5
			},
			priceGovStable: sdk.NewDec(5),
			accFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.CollDenom, 1_000_000_000),
			),

			// Since 'neededCollUSD' is
			expectedNeededUSD: sdk.MustNewDecFromStr("150000.5"),
			msg: types.MsgRecollateralize{
				Creator: sample.AccAddress().String(),
				Coll:    sdk.NewCoin(common.CollDenom, sdk.NewInt(50_000)),
			},
			response: &types.MsgRecollateralizeResponse{
				/*
					Gov.Amount = inCollUSD * (1 + bonusRate) / priceGovStable
					  = msg.Coll.Amount * priceCollStable (1.002) / priceGovStable
					  = 50_000 * 1.099999 * (1.002) / priceGovStable
					  = 55109.9499 / priceGovStable
					  = 11021.98998 -> 11_021
				*/
				Gov: sdk.NewCoin(common.GovDenom, sdk.NewInt(11_021)),
			},
			expectedPass: true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			require.EqualValues(t, tc.expectedNeededUSD, tc.scenario.CalcNeededUSD())

			matrixApp, ctx := testutil.NewMatrixApp(true)
			stablecoinKeeper := &matrixApp.StablecoinKeeper
			stablecoinKeeper.SetCollRatio(ctx, tc.scenario.collRatio)
			matrixApp.BankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(
					sdk.NewCoin(common.CollDenom, tc.scenario.protocolColl),
					sdk.NewCoin(common.StableDenom, tc.scenario.stableSupply),
				),
			)
			// Fund account
			caller, err := sdk.AccAddressFromBech32(tc.msg.Creator)
			if tc.expectedPass {
				require.NoError(t, err)
			}
			err = simapp.FundAccount(matrixApp.BankKeeper, ctx, caller, tc.accFunds)
			if tc.expectedPass {
				require.NoError(t, err)
			}

			// Set up markets for the pricefeed keeper.
			oracle := sample.AccAddress()
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			pricefeedParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: common.CollStablePool, BaseAsset: common.CollDenom,
						QuoteAsset: common.StableDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
					{MarketID: common.GovStablePool, BaseAsset: common.GovDenom,
						QuoteAsset: common.StableDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
				}}
			matrixApp.PriceKeeper.SetParams(ctx, pricefeedParams)

			// Post prices to each specified market with the oracle.
			prices := map[string]sdk.Dec{
				common.CollStablePool: tc.scenario.priceCollStable,
				common.GovStablePool:  tc.priceGovStable,
			}
			for _, marketID := range tc.postedMarketIDs {
				_, err := matrixApp.PriceKeeper.SetPrice(
					ctx, oracle, marketID, prices[marketID], priceExpiry)
				require.NoError(t, err)

				// Update the 'CurrentPrice' posted by the oracles.
				err = matrixApp.PriceKeeper.SetCurrentPrices(ctx, marketID)
				require.NoError(t, err, "Error posting price for market: %d", marketID)
			}

			goCtx := sdk.WrapSDKContext(ctx)
			response, err := stablecoinKeeper.Recollateralize(goCtx, &tc.msg)
			if tc.expectedPass {
				require.NoError(t, err)
				require.EqualValues(t, tc.response, response)
			} else {
				require.Error(t, err)
			}
		},
		)
	}

}

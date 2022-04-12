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
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------------
// MintStable
// ------------------------------------------------------------------

func TestMsgMint_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name string
		msg  types.MsgMintStable
		err  error
	}{
		{
			name: "invalid address",
			msg: types.MsgMintStable{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
			},
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgMintStableResponse_Supply(t *testing.T) {
	accFundsGovAmount := sdk.NewCoin(common.GovDenom, sdk.NewInt(10_000))
	accFundsCollAmount := sdk.NewCoin(common.CollDenom, sdk.NewInt(900_000))
	neededGovFees := sdk.NewCoin(common.GovDenom, sdk.NewInt(20))      // 0.002 fee
	neededCollFees := sdk.NewCoin(common.CollDenom, sdk.NewInt(1_800)) // 0.002 fee

	accFundsAmt := sdk.NewCoins(
		accFundsGovAmount.Add(neededGovFees),
		accFundsCollAmount.Add(neededCollFees),
	)

	tests := []struct {
		name        string
		accFunds    sdk.Coins
		msgMint     types.MsgMintStable
		msgResponse types.MsgMintStableResponse
		govPrice    sdk.Dec
		collPrice   sdk.Dec
		supplyMtrx  sdk.Coin
		supplyUsdm  sdk.Coin
		err         error
	}{
		{
			name:     "Successful mint",
			accFunds: accFundsAmt,
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(1_000_000)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable:    sdk.NewCoin(common.StableDenom, sdk.NewInt(1_000_000)),
				UsedCoins: sdk.NewCoins(accFundsCollAmount, accFundsGovAmount),
				FeesPayed: sdk.NewCoins(neededCollFees, neededGovFees),
			},
			govPrice:   sdk.MustNewDecFromStr("10"),
			collPrice:  sdk.MustNewDecFromStr("1"),
			supplyMtrx: sdk.NewCoin(common.GovDenom, sdk.NewInt(10)),
			// 10_000 - 20 (neededAmt - fees) - 10 (0.5 of fees from EFund are burned)
			supplyUsdm: sdk.NewCoin(common.StableDenom, sdk.NewInt(1_000_000)),
			err:        nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			acc, _ := sdk.AccAddressFromBech32(tc.msgMint.Creator)
			oracle := sample.AccAddress()

			// We get module account, to create it.
			matrixApp.AccountKeeper.GetModuleAccount(ctx, types.StableEFModuleAccount)

			// Set up markets for the pricefeed keeper.
			priceKeeper := &matrixApp.PriceKeeper
			pfParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: common.GovCollPool, BaseAsset: common.CollDenom,
						QuoteAsset: common.GovDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
					{MarketID: common.CollStablePool, BaseAsset: common.CollDenom,
						QuoteAsset: common.StableDenom,
						Oracles:    []sdk.AccAddress{oracle}, Active: true},
				}}
			priceKeeper.SetParams(ctx, pfParams)

			collRatio := sdk.MustNewDecFromStr("0.9")
			feeRatio := sdk.MustNewDecFromStr("0.002")
			feeRatioEF := sdk.MustNewDecFromStr("0.5")
			matrixApp.StablecoinKeeper.SetParams(
				ctx, types.NewParams(collRatio, feeRatio, feeRatioEF))

			// Post prices to each market with the oracle.
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			_, err := priceKeeper.SetPrice(
				ctx, oracle, common.GovCollPool, tc.govPrice, priceExpiry,
			)
			require.NoError(t, err)
			_, err = priceKeeper.SetPrice(
				ctx, oracle, common.CollStablePool, tc.collPrice, priceExpiry,
			)
			require.NoError(t, err)

			// Update the 'CurrentPrice' posted by the oracles.
			for _, market := range pfParams.Markets {
				err = priceKeeper.SetCurrentPrices(ctx, market.MarketID)
				require.NoError(t, err, "Error posting price for market: %d", market)
			}

			// Fund account
			err = simapp.FundAccount(matrixApp.BankKeeper, ctx, acc, tc.accFunds)
			require.NoError(t, err)

			// Mint USDM -> Response contains Stable (sdk.Coin)
			goCtx := sdk.WrapSDKContext(ctx)
			mintStableResponse, err := matrixApp.StablecoinKeeper.MintStable(
				goCtx, &tc.msgMint)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, *mintStableResponse, tc.msgResponse, "mintStableResponse")

			require.Equal(t, matrixApp.StablecoinKeeper.GetSupplyMTRX(ctx), tc.supplyMtrx)
			require.Equal(t, matrixApp.StablecoinKeeper.GetSupplyUSDM(ctx), tc.supplyUsdm)

			// Check balances in EF
			efModuleBalance := matrixApp.BankKeeper.GetAllBalances(ctx, matrixApp.AccountKeeper.GetModuleAddress(types.StableEFModuleAccount))
			collFeesInEf := neededCollFees.Amount.ToDec().Mul(sdk.MustNewDecFromStr("0.5")).TruncateInt()
			require.Equal(t, sdk.NewCoins(sdk.NewCoin(common.CollDenom, collFeesInEf)), efModuleBalance)

			// Check balances in Treasury
			treasuryModuleBalance := matrixApp.BankKeeper.
				GetAllBalances(ctx, matrixApp.AccountKeeper.GetModuleAddress(common.TreasuryPoolModuleAccount))
			collFeesInTreasury := neededCollFees.Amount.ToDec().Mul(sdk.MustNewDecFromStr("0.5")).TruncateInt()
			govFeesInTreasury := neededGovFees.Amount.ToDec().Mul(sdk.MustNewDecFromStr("0.5")).TruncateInt()
			require.Equal(
				t,
				sdk.NewCoins(
					sdk.NewCoin(common.CollDenom, collFeesInTreasury),
					sdk.NewCoin(common.GovDenom, govFeesInTreasury),
				),
				treasuryModuleBalance,
			)
		})
	}

}

func TestMsgMintStableResponse_NotEnoughFunds(t *testing.T) {

	testCases := []struct {
		name        string
		accFunds    sdk.Coins
		msgMint     types.MsgMintStable
		msgResponse types.MsgMintStableResponse
		govPrice    sdk.Dec
		collPrice   sdk.Dec
		err         error
	}{
		{
			name: "User has no GOV",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(9001)),
				sdk.NewCoin(common.GovDenom, sdk.NewInt(0)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.StableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       types.NotEnoughBalance.Wrap(common.GovDenom),
		}, {
			name: "User has no COLL",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(0)),
				sdk.NewCoin(common.GovDenom, sdk.NewInt(9001)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.StableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       types.NotEnoughBalance.Wrap(common.CollDenom),
		},
		{
			name: "Not enough GOV",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(9001)),
				sdk.NewCoin(common.GovDenom, sdk.NewInt(1)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(1000)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.StableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err: types.NotEnoughBalance.Wrap(
				sdk.NewCoin(common.GovDenom, sdk.NewInt(1)).String()),
		}, {
			name: "Not enough COLL",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(1)),
				sdk.NewCoin(common.GovDenom, sdk.NewInt(9001)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.StableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err: types.NotEnoughBalance.Wrap(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(1)).String()),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			matrixApp, ctx := testutil.NewMatrixApp()
			acc, _ := sdk.AccAddressFromBech32(tc.msgMint.Creator)
			oracle := sample.AccAddress()

			// We get module account, to create it.
			matrixApp.AccountKeeper.GetModuleAccount(ctx, types.StableEFModuleAccount)

			// Set up markets for the pricefeed keeper.
			priceKeeper := &matrixApp.PriceKeeper
			pfParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: common.GovCollPool, BaseAsset: common.CollDenom, QuoteAsset: common.GovDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
					{MarketID: common.CollStablePool, BaseAsset: common.CollDenom, QuoteAsset: common.StableDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
				}}
			priceKeeper.SetParams(ctx, pfParams)

			collRatio := sdk.MustNewDecFromStr("0.9")
			feeRatio := sdk.ZeroDec()
			feeRatioEF := sdk.MustNewDecFromStr("0.5")
			matrixApp.StablecoinKeeper.SetParams(
				ctx, types.NewParams(collRatio, feeRatio, feeRatioEF))

			// Post prices to each market with the oracle.
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			_, err := priceKeeper.SetPrice(
				ctx, oracle, common.GovCollPool, tc.govPrice, priceExpiry,
			)
			require.NoError(t, err)
			_, err = priceKeeper.SetPrice(
				ctx, oracle, common.CollStablePool, tc.collPrice, priceExpiry,
			)
			require.NoError(t, err)

			// Update the 'CurrentPrice' posted by the oracles.
			for _, market := range pfParams.Markets {
				err = priceKeeper.SetCurrentPrices(ctx, market.MarketID)
				require.NoError(t, err, "Error posting price for market: %d", market)
			}

			// Fund account
			err = simapp.FundAccount(matrixApp.BankKeeper, ctx, acc, tc.accFunds)
			require.NoError(t, err)

			// Mint USDM -> Response contains Stable (sdk.Coin)
			goCtx := sdk.WrapSDKContext(ctx)
			mintStableResponse, err := matrixApp.StablecoinKeeper.MintStable(
				goCtx, &tc.msgMint)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, *mintStableResponse, tc.msgResponse, "mintStableResponse")

			balances := matrixApp.BankKeeper.GetAllBalances(ctx, matrixApp.AccountKeeper.GetModuleAddress(types.StableEFModuleAccount))
			require.Equal(t, mintStableResponse.FeesPayed, balances)
		})
	}
}

// ------------------------------------------------------------------
// BurnStable / Redeem
// ------------------------------------------------------------------

func TestMsgBurn_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name string
		msg  types.MsgBurnStable
		err  error
	}{
		{
			name: "invalid address",
			msg: types.MsgBurnStable{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: types.MsgBurnStable{
				Creator: sample.AccAddress().String(),
			},
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgBurnResponse_NotEnoughFunds(t *testing.T) {

	type TestCase struct {
		name         string
		accFunds     sdk.Coins
		moduleFunds  sdk.Coins
		msgBurn      types.MsgBurnStable
		msgResponse  types.MsgBurnStableResponse
		govPrice     sdk.Dec
		collPrice    sdk.Dec
		expectedPass bool
		err          string
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			acc, _ := sdk.AccAddressFromBech32(tc.msgBurn.Creator)
			oracle := sample.AccAddress()

			// Set up markets for the pricefeed keeper.
			priceKeeper := &matrixApp.PriceKeeper
			pfParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: common.GovCollPool, BaseAsset: common.CollDenom, QuoteAsset: common.GovDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
					{MarketID: common.CollStablePool, BaseAsset: common.CollDenom, QuoteAsset: common.StableDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
				}}
			priceKeeper.SetParams(ctx, pfParams)

			// Post prices to each market with the oracle.
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			_, err := priceKeeper.SetPrice(
				ctx, oracle, common.GovCollPool, tc.govPrice, priceExpiry,
			)
			require.NoError(t, err)
			_, err = priceKeeper.SetPrice(
				ctx, oracle, common.CollStablePool, tc.collPrice, priceExpiry,
			)
			require.NoError(t, err)

			// Update the 'CurrentPrice' posted by the oracles.
			for _, market := range pfParams.Markets {
				err = priceKeeper.SetCurrentPrices(ctx, market.MarketID)
				require.NoError(t, err, "Error posting price for market: %d", market)
			}

			// Add collaterals to the module
			err = matrixApp.BankKeeper.MintCoins(ctx, types.ModuleName, tc.moduleFunds)
			if err != nil {
				panic(err)
			}

			err = simapp.FundAccount(matrixApp.BankKeeper, ctx, acc, tc.accFunds)
			require.NoError(t, err)

			// Burn USDM -> Response contains GOV and COLL
			goCtx := sdk.WrapSDKContext(ctx)
			burnStableResponse, err := matrixApp.StablecoinKeeper.BurnStable(
				goCtx, &tc.msgBurn)

			if !tc.expectedPass {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)

				return
			}
			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, burnStableResponse, &tc.msgResponse, "burnStableResponse")
		})
	}

	testCases := []TestCase{
		{
			name:     "Not enough stable",
			accFunds: sdk.NewCoins(sdk.NewInt64Coin(common.StableDenom, 10)),
			msgBurn: types.MsgBurnStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewInt64Coin(common.StableDenom, 9001),
			},
			msgResponse: types.MsgBurnStableResponse{
				Collateral: sdk.NewCoin(common.GovDenom, sdk.ZeroInt()),
				Gov:        sdk.NewCoin(common.CollDenom, sdk.ZeroInt()),
			},
			govPrice:     sdk.MustNewDecFromStr("10"),
			collPrice:    sdk.MustNewDecFromStr("1"),
			expectedPass: false,
			err:          "insufficient funds",
		},
		{
			name:      "Stable is zero",
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			accFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.StableDenom, 1000000000),
			),
			moduleFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.CollDenom, 100000000),
			),
			msgBurn: types.MsgBurnStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.ZeroInt()),
			},
			msgResponse: types.MsgBurnStableResponse{
				Gov:        sdk.NewCoin(common.GovDenom, sdk.ZeroInt()),
				Collateral: sdk.NewCoin(common.CollDenom, sdk.ZeroInt()),
			},
			expectedPass: true,
			err:          types.NoCoinFound.Wrap(common.StableDenom).Error(),
		},
	}
	for _, test := range testCases {
		executeTest(t, test)
	}
}

func TestMsgBurnResponse_EnoughFunds(t *testing.T) {

	type TestCase struct {
		name         string
		accFunds     sdk.Coins
		moduleFunds  sdk.Coins
		msgBurn      types.MsgBurnStable
		msgResponse  types.MsgBurnStableResponse
		govPrice     sdk.Dec
		collPrice    sdk.Dec
		expectedPass bool
		err          string
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			acc, _ := sdk.AccAddressFromBech32(tc.msgBurn.Creator)
			oracle := sample.AccAddress()

			// Set up markets for the pricefeed keeper.
			priceKeeper := &matrixApp.PriceKeeper
			pfParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: common.GovCollPool, BaseAsset: common.CollDenom, QuoteAsset: common.GovDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
					{MarketID: common.CollStablePool, BaseAsset: common.CollDenom, QuoteAsset: common.StableDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
				}}
			priceKeeper.SetParams(ctx, pfParams)

			// Post prices to each market with the oracle.
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			_, err := priceKeeper.SetPrice(
				ctx, oracle, common.GovCollPool, tc.govPrice, priceExpiry,
			)
			require.NoError(t, err)
			_, err = priceKeeper.SetPrice(
				ctx, oracle, common.CollStablePool, tc.collPrice, priceExpiry,
			)
			require.NoError(t, err)

			// Update the 'CurrentPrice' posted by the oracles.
			for _, market := range pfParams.Markets {
				err = priceKeeper.SetCurrentPrices(ctx, market.MarketID)
				require.NoError(t, err, "Error posting price for market: %d", market)
			}

			// Add collaterals to the module
			err = matrixApp.BankKeeper.MintCoins(ctx, types.ModuleName, tc.moduleFunds)
			if err != nil {
				panic(err)
			}

			err = simapp.FundAccount(matrixApp.BankKeeper, ctx, acc, tc.accFunds)
			require.NoError(t, err)

			// Burn USDM -> Response contains GOV and COLL
			goCtx := sdk.WrapSDKContext(ctx)
			burnStableResponse, err := matrixApp.StablecoinKeeper.BurnStable(
				goCtx, &tc.msgBurn)

			if !tc.expectedPass {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)

				return
			}
			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, burnStableResponse, &tc.msgResponse, "burnStableResponse")
		})
	}

	testCases := []TestCase{
		{
			name:      "Happy path",
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			accFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.StableDenom, 1000000000),
			),
			moduleFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.CollDenom, 100000000),
			),
			msgBurn: types.MsgBurnStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewInt64Coin(common.StableDenom, 10000000),
			},
			msgResponse: types.MsgBurnStableResponse{
				Gov:        sdk.NewInt64Coin(common.GovDenom, 100000),
				Collateral: sdk.NewInt64Coin(common.CollDenom, 9000000),
			},
			expectedPass: true,
		},
	}
	for _, test := range testCases {
		executeTest(t, test)
	}
}

func TestMsgBurnResponse_supply(t *testing.T) {

	type TestCase struct {
		name         string
		accFunds     sdk.Coins
		moduleFunds  sdk.Coins
		msgBurn      types.MsgBurnStable
		msgResponse  types.MsgBurnStableResponse
		govPrice     sdk.Dec
		collPrice    sdk.Dec
		supplyMtrx   sdk.Coin
		supplyUsdm   sdk.Coin
		expectedPass bool
		err          string
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			acc, _ := sdk.AccAddressFromBech32(tc.msgBurn.Creator)
			oracle := sample.AccAddress()

			// Set up markets for the pricefeed keeper.
			priceKeeper := &matrixApp.PriceKeeper
			pfParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: common.GovCollPool, BaseAsset: common.CollDenom, QuoteAsset: common.GovDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
					{MarketID: common.CollStablePool, BaseAsset: common.CollDenom, QuoteAsset: common.StableDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
				}}
			priceKeeper.SetParams(ctx, pfParams)

			// Post prices to each market with the oracle.
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			_, err := priceKeeper.SetPrice(
				ctx, oracle, common.GovCollPool, tc.govPrice, priceExpiry,
			)
			require.NoError(t, err)
			_, err = priceKeeper.SetPrice(
				ctx, oracle, common.CollStablePool, tc.collPrice, priceExpiry,
			)
			require.NoError(t, err)

			// Update the 'CurrentPrice' posted by the oracles.
			for _, market := range pfParams.Markets {
				err = priceKeeper.SetCurrentPrices(ctx, market.MarketID)
				require.NoError(t, err, "Error posting price for market: %d", market)
			}

			// Add collaterals to the module
			err = matrixApp.BankKeeper.MintCoins(ctx, types.ModuleName, tc.moduleFunds)
			if err != nil {
				panic(err)
			}

			err = simapp.FundAccount(matrixApp.BankKeeper, ctx, acc, tc.accFunds)
			require.NoError(t, err)

			// Burn USDM -> Response contains GOV and COLL
			goCtx := sdk.WrapSDKContext(ctx)
			burnStableResponse, err := matrixApp.StablecoinKeeper.BurnStable(
				goCtx, &tc.msgBurn)

			if !tc.expectedPass {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)

				return
			}
			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, burnStableResponse, &tc.msgResponse, "burnStableResponse")

			require.Equal(t, matrixApp.StablecoinKeeper.GetSupplyMTRX(ctx), tc.supplyMtrx)
			require.Equal(t, matrixApp.StablecoinKeeper.GetSupplyUSDM(ctx), tc.supplyUsdm)
		})
	}

	testCases := []TestCase{
		{
			name:      "Happy path",
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			accFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.StableDenom, 1000000000),
			),
			moduleFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.CollDenom, 100000000),
			),
			msgBurn: types.MsgBurnStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewInt64Coin(common.StableDenom, 10000000),
			},
			msgResponse: types.MsgBurnStableResponse{
				Gov:        sdk.NewInt64Coin(common.GovDenom, 100000),
				Collateral: sdk.NewInt64Coin(common.CollDenom, 9000000),
			},
			supplyMtrx:   sdk.NewCoin(common.GovDenom, sdk.NewInt(100000)),
			supplyUsdm:   sdk.NewCoin(common.StableDenom, sdk.NewInt(1000000000-10000000)),
			expectedPass: true,
		},
	}
	for _, test := range testCases {
		executeTest(t, test)
	}
}

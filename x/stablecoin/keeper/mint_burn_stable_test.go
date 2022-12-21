package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/testutil"

	simapp2 "github.com/NibiruChain/nibiru/simapp"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
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
				Creator: testutil.AccAddress().String(),
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

func TestMsgMintStableResponse_HappyPath(t *testing.T) {
	accFundsGovAmount := sdk.NewCoin(common.DenomNIBI, sdk.NewInt(10_000))
	accFundsCollAmount := sdk.NewCoin(common.DenomUSDC, sdk.NewInt(900_000))
	neededGovFees := sdk.NewCoin(common.DenomNIBI, sdk.NewInt(20))     // 0.002 fee
	neededCollFees := sdk.NewCoin(common.DenomUSDC, sdk.NewInt(1_800)) // 0.002 fee

	accFundsAmt := sdk.NewCoins(
		accFundsGovAmount.Add(neededGovFees),
		accFundsCollAmount.Add(neededCollFees),
	)

	tests := []struct {
		name                   string
		accFunds               sdk.Coins
		msgMint                types.MsgMintStable
		msgResponse            types.MsgMintStableResponse
		govPrice               sdk.Dec
		collPrice              sdk.Dec
		supplyNIBI             sdk.Coin
		supplyNUSD             sdk.Coin
		err                    error
		isCollateralRatioValid bool
	}{
		{
			name:     "Not able to mint because of no posted prices",
			accFunds: accFundsAmt,
			msgMint: types.MsgMintStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewCoin(common.DenomNUSD, sdk.NewInt(1*common.Precision)),
			},
			govPrice:               sdk.MustNewDecFromStr("10"),
			collPrice:              sdk.MustNewDecFromStr("1"),
			err:                    types.NoValidCollateralRatio,
			isCollateralRatioValid: false,
		},
		{
			name:     "Successful mint",
			accFunds: accFundsAmt,
			msgMint: types.MsgMintStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewCoin(common.DenomNUSD, sdk.NewInt(1*common.Precision)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable:    sdk.NewCoin(common.DenomNUSD, sdk.NewInt(1*common.Precision)),
				UsedCoins: sdk.NewCoins(accFundsCollAmount, accFundsGovAmount),
				FeesPayed: sdk.NewCoins(neededCollFees, neededGovFees),
			},
			govPrice:   sdk.MustNewDecFromStr("10"),
			collPrice:  sdk.MustNewDecFromStr("1"),
			supplyNIBI: sdk.NewCoin(common.DenomNIBI, sdk.NewInt(10)),
			// 10_000 - 20 (neededAmt - fees) - 10 (0.5 of fees from EFund are burned)
			supplyNUSD:             sdk.NewCoin(common.DenomNUSD, sdk.NewInt(1*common.Precision)),
			err:                    nil,
			isCollateralRatioValid: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
			acc, _ := sdk.AccAddressFromBech32(tc.msgMint.Creator)

			// We get module account, to create it.
			nibiruApp.AccountKeeper.GetModuleAccount(ctx, types.StableEFModuleAccount)

			// Set up pairs for the oracle keeper.

			collRatio := sdk.MustNewDecFromStr("0.9")
			feeRatio := sdk.MustNewDecFromStr("0.002")
			feeRatioEF := sdk.MustNewDecFromStr("0.5")
			bonusRateRecoll := sdk.MustNewDecFromStr("0.002")
			adjustmentStep := sdk.MustNewDecFromStr("0.0025")
			priceLowerBound := sdk.MustNewDecFromStr("0.9999")
			priceUpperBound := sdk.MustNewDecFromStr("1.0001")

			nibiruApp.StablecoinKeeper.SetParams(
				ctx, types.NewParams(
					collRatio,
					feeRatio,
					feeRatioEF,
					bonusRateRecoll,
					"15 min",
					adjustmentStep,
					priceLowerBound,
					priceUpperBound,
					tc.isCollateralRatioValid,
				),
			)

			// Post prices to each pair with the oracle.
			nibiruApp.OracleKeeper.SetPrice(ctx, common.Pair_NIBI_NUSD.String(), tc.govPrice)
			nibiruApp.OracleKeeper.SetPrice(ctx, common.Pair_USDC_NUSD.String(), tc.collPrice)

			// Fund account
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, acc, tc.accFunds))

			// Mint NUSD -> Response contains Stable (sdk.Coin)
			goCtx := sdk.WrapSDKContext(ctx)
			mintStableResponse, err := nibiruApp.StablecoinKeeper.MintStable(
				goCtx, &tc.msgMint)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)
			assert.EqualValues(t, tc.msgResponse, *mintStableResponse)
			assert.Equal(t, nibiruApp.StablecoinKeeper.GetSupplyNIBI(ctx), tc.supplyNIBI)
			assert.Equal(t, nibiruApp.StablecoinKeeper.GetSupplyNUSD(ctx), tc.supplyNUSD)

			// Check balances in EF
			efModuleBalance := nibiruApp.BankKeeper.GetAllBalances(
				ctx, nibiruApp.AccountKeeper.GetModuleAddress(types.StableEFModuleAccount),
			)
			collFeesInEf := neededCollFees.Amount.ToDec().Mul(sdk.MustNewDecFromStr("0.5")).TruncateInt()
			assert.Equal(t, sdk.NewCoins(sdk.NewCoin(common.DenomUSDC, collFeesInEf)), efModuleBalance)

			// Check balances in Treasury
			treasuryModuleBalance := nibiruApp.BankKeeper.
				GetAllBalances(ctx, nibiruApp.AccountKeeper.GetModuleAddress(common.TreasuryPoolModuleAccount))
			collFeesInTreasury := neededCollFees.Amount.ToDec().Mul(sdk.MustNewDecFromStr("0.5")).TruncateInt()
			govFeesInTreasury := neededGovFees.Amount.ToDec().Mul(sdk.MustNewDecFromStr("0.5")).TruncateInt()
			assert.Equal(
				t,
				sdk.NewCoins(
					sdk.NewCoin(common.DenomUSDC, collFeesInTreasury),
					sdk.NewCoin(common.DenomNIBI, govFeesInTreasury),
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
				sdk.NewCoin(common.DenomUSDC, sdk.NewInt(9001)),
				sdk.NewCoin(common.DenomNIBI, sdk.NewInt(0)),
			),
			msgMint: types.MsgMintStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewCoin(common.DenomNUSD, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.DenomNUSD, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       types.NotEnoughBalance.Wrap(common.DenomNIBI),
		}, {
			name: "User has no COLL",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.DenomUSDC, sdk.NewInt(0)),
				sdk.NewCoin(common.DenomNIBI, sdk.NewInt(9001)),
			),
			msgMint: types.MsgMintStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewCoin(common.DenomNUSD, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.DenomNUSD, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       types.NotEnoughBalance.Wrap(common.DenomUSDC),
		},
		{
			name: "Not enough GOV",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.DenomUSDC, sdk.NewInt(9001)),
				sdk.NewCoin(common.DenomNIBI, sdk.NewInt(1)),
			),
			msgMint: types.MsgMintStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewCoin(common.DenomNUSD, sdk.NewInt(1000)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.DenomNUSD, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err: types.NotEnoughBalance.Wrap(
				sdk.NewCoin(common.DenomNIBI, sdk.NewInt(1)).String()),
		}, {
			name: "Not enough COLL",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.DenomUSDC, sdk.NewInt(1)),
				sdk.NewCoin(common.DenomNIBI, sdk.NewInt(9001)),
			),
			msgMint: types.MsgMintStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewCoin(common.DenomNUSD, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.DenomNUSD, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err: types.NotEnoughBalance.Wrap(
				sdk.NewCoin(common.DenomUSDC, sdk.NewInt(1)).String()),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
			acc, _ := sdk.AccAddressFromBech32(tc.msgMint.Creator)

			// We get module account, to create it.
			nibiruApp.AccountKeeper.GetModuleAccount(ctx, types.StableEFModuleAccount)

			collRatio := sdk.MustNewDecFromStr("0.9")
			feeRatio := sdk.ZeroDec()
			feeRatioEF := sdk.MustNewDecFromStr("0.5")
			bonusRateRecoll := sdk.MustNewDecFromStr("0.002")
			adjustmentStep := sdk.MustNewDecFromStr("0.0025")
			priceLowerBound := sdk.MustNewDecFromStr("0.9999")
			priceUpperBound := sdk.MustNewDecFromStr("1.0001")

			nibiruApp.StablecoinKeeper.SetParams(
				ctx, types.NewParams(
					collRatio,
					feeRatio,
					feeRatioEF,
					bonusRateRecoll,
					"15 min",
					adjustmentStep,
					priceLowerBound,
					priceUpperBound,
					true,
				),
			)

			t.Log("Post prices to each pair with the oracle.")
			nibiruApp.OracleKeeper.SetPrice(ctx, common.Pair_NIBI_NUSD.String(), tc.govPrice)
			nibiruApp.OracleKeeper.SetPrice(ctx, common.Pair_USDC_NUSD.String(), tc.collPrice)

			// Fund account
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, acc, tc.accFunds))

			// Mint NUSD -> Response contains Stable (sdk.Coin)
			goCtx := sdk.WrapSDKContext(ctx)
			mintStableResponse, err := nibiruApp.StablecoinKeeper.MintStable(
				goCtx, &tc.msgMint)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)
			assert.EqualValues(t, tc.msgResponse, *mintStableResponse)

			balances := nibiruApp.BankKeeper.GetAllBalances(ctx, nibiruApp.AccountKeeper.GetModuleAddress(types.StableEFModuleAccount))
			assert.Equal(t, mintStableResponse.FeesPayed, balances)
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
				Creator: testutil.AccAddress().String(),
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
	tests := []struct {
		name         string
		accFunds     sdk.Coins
		moduleFunds  sdk.Coins
		msgBurn      types.MsgBurnStable
		msgResponse  *types.MsgBurnStableResponse
		govPrice     sdk.Dec
		collPrice    sdk.Dec
		expectedPass bool
		err          string
	}{
		{
			name:     "Not enough stable",
			accFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomNUSD, 10)),
			msgBurn: types.MsgBurnStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewInt64Coin(common.DenomNUSD, 9001),
			},
			msgResponse: &types.MsgBurnStableResponse{
				Collateral: sdk.NewCoin(common.DenomNIBI, sdk.ZeroInt()),
				Gov:        sdk.NewCoin(common.DenomUSDC, sdk.ZeroInt()),
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
				sdk.NewInt64Coin(common.DenomNUSD, 1000*common.Precision),
			),
			moduleFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.DenomUSDC, 100*common.Precision),
			),
			msgBurn: types.MsgBurnStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewCoin(common.DenomNUSD, sdk.ZeroInt()),
			},
			msgResponse: &types.MsgBurnStableResponse{
				Gov:        sdk.NewCoin(common.DenomNIBI, sdk.ZeroInt()),
				Collateral: sdk.NewCoin(common.DenomUSDC, sdk.ZeroInt()),
				FeesPayed:  sdk.NewCoins(),
			},
			expectedPass: true,
			err:          types.NoCoinFound.Wrap(common.DenomNUSD).Error(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
			acc, _ := sdk.AccAddressFromBech32(tc.msgBurn.Creator)

			// Set stablecoin params
			collRatio := sdk.MustNewDecFromStr("0.9")
			feeRatio := sdk.MustNewDecFromStr("0.002")
			feeRatioEF := sdk.MustNewDecFromStr("0.5")
			bonusRateRecoll := sdk.MustNewDecFromStr("0.002")
			adjustmentStep := sdk.MustNewDecFromStr("0.0025")
			priceLowerBound := sdk.MustNewDecFromStr("0.9999")
			priceUpperBound := sdk.MustNewDecFromStr("1.0001")

			nibiruApp.StablecoinKeeper.SetParams(
				ctx, types.NewParams(
					collRatio,
					feeRatio,
					feeRatioEF,
					bonusRateRecoll,
					"15 min",
					adjustmentStep,
					priceLowerBound,
					priceUpperBound,
					true,
				),
			)

			defaultParams := types.DefaultParams()
			defaultParams.IsCollateralRatioValid = true
			nibiruApp.StablecoinKeeper.SetParams(ctx, defaultParams)

			t.Log("Post prices to each pair with the oracle.")
			nibiruApp.OracleKeeper.SetPrice(ctx, common.Pair_NIBI_NUSD.String(), tc.govPrice)
			nibiruApp.OracleKeeper.SetPrice(ctx, common.Pair_USDC_NUSD.String(), tc.collPrice)

			// Add collaterals to the module
			require.NoError(t, nibiruApp.BankKeeper.MintCoins(ctx, types.ModuleName, tc.moduleFunds))
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, acc, tc.accFunds))

			// Burn NUSD -> Response contains GOV and COLL
			goCtx := sdk.WrapSDKContext(ctx)
			burnStableResponse, err := nibiruApp.StablecoinKeeper.BurnStable(
				goCtx, &tc.msgBurn)

			if !tc.expectedPass {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)

				return
			}
			require.NoError(t, err)
			assert.EqualValues(t, tc.msgResponse, burnStableResponse)
		})
	}
}

func TestMsgBurnResponse_HappyPath(t *testing.T) {
	tests := []struct {
		name                   string
		accFunds               sdk.Coins
		moduleFunds            sdk.Coins
		msgBurn                types.MsgBurnStable
		msgResponse            types.MsgBurnStableResponse
		govPrice               sdk.Dec
		collPrice              sdk.Dec
		supplyNIBI             sdk.Coin
		supplyNUSD             sdk.Coin
		ecosystemFund          sdk.Coins
		treasuryFund           sdk.Coins
		expectedPass           bool
		err                    error
		isCollateralRatioValid bool
	}{
		{
			name:      "invalid collateral ratio",
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			accFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.DenomNUSD, 1_000*common.Precision),
			),
			moduleFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.DenomUSDC, 100*common.Precision),
			),
			msgBurn: types.MsgBurnStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewInt64Coin(common.DenomNUSD, 10*common.Precision),
			},
			ecosystemFund:          sdk.NewCoins(sdk.NewInt64Coin(common.DenomUSDC, 9000)),
			treasuryFund:           sdk.NewCoins(sdk.NewInt64Coin(common.DenomUSDC, 9000), sdk.NewInt64Coin(common.DenomNIBI, 100)),
			expectedPass:           false,
			isCollateralRatioValid: false,
			err:                    types.NoValidCollateralRatio,
		},
		{
			name:      "Happy path",
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			accFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.DenomNUSD, 1_000*common.Precision),
			),
			moduleFunds: sdk.NewCoins(
				sdk.NewInt64Coin(common.DenomUSDC, 100*common.Precision),
			),
			msgBurn: types.MsgBurnStable{
				Creator: testutil.AccAddress().String(),
				Stable:  sdk.NewInt64Coin(common.DenomNUSD, 10*common.Precision),
			},
			msgResponse: types.MsgBurnStableResponse{
				Gov:        sdk.NewInt64Coin(common.DenomNIBI, 100_000-200),               // amount - fees 0,02%
				Collateral: sdk.NewInt64Coin(common.DenomUSDC, 9*common.Precision-18_000), // amount - fees 0,02%
				FeesPayed: sdk.NewCoins(
					sdk.NewInt64Coin(common.DenomNIBI, 200),
					sdk.NewInt64Coin(common.DenomUSDC, 18_000),
				),
			},
			supplyNIBI:             sdk.NewCoin(common.DenomNIBI, sdk.NewInt(100_000-100)), // nibiru minus 0.5 of fees burned (the part that goes to EF)
			supplyNUSD:             sdk.NewCoin(common.DenomNUSD, sdk.NewInt(1_000*common.Precision-10*common.Precision)),
			ecosystemFund:          sdk.NewCoins(sdk.NewInt64Coin(common.DenomUSDC, 9000)),
			treasuryFund:           sdk.NewCoins(sdk.NewInt64Coin(common.DenomUSDC, 9000), sdk.NewInt64Coin(common.DenomNIBI, 100)),
			expectedPass:           true,
			isCollateralRatioValid: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
			acc, _ := sdk.AccAddressFromBech32(tc.msgBurn.Creator)

			// Set stablecoin params
			collRatio := sdk.MustNewDecFromStr("0.9")
			feeRatio := sdk.MustNewDecFromStr("0.002")
			feeRatioEF := sdk.MustNewDecFromStr("0.5")
			bonusRateRecoll := sdk.MustNewDecFromStr("0.002")
			adjustmentStep := sdk.MustNewDecFromStr("0.0025")
			priceLowerBound := sdk.MustNewDecFromStr("0.9999")
			priceUpperBound := sdk.MustNewDecFromStr("1.0001")

			nibiruApp.StablecoinKeeper.SetParams(
				ctx, types.NewParams(
					collRatio,
					feeRatio,
					feeRatioEF,
					bonusRateRecoll,
					"15 min",
					adjustmentStep,
					priceLowerBound,
					priceUpperBound,
					tc.isCollateralRatioValid,
				),
			)

			nibiruApp.OracleKeeper.SetPrice(ctx, common.Pair_NIBI_NUSD.String(), tc.govPrice)
			nibiruApp.OracleKeeper.SetPrice(ctx, common.Pair_USDC_NUSD.String(), tc.collPrice)

			// Add collaterals to the module
			require.NoError(t, nibiruApp.BankKeeper.MintCoins(ctx, types.ModuleName, tc.moduleFunds))
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, acc, tc.accFunds))

			// Burn NUSD -> Response contains GOV and COLL
			goCtx := sdk.WrapSDKContext(ctx)
			burnStableResponse, err := nibiruApp.StablecoinKeeper.BurnStable(
				goCtx, &tc.msgBurn)

			if !tc.expectedPass {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)
			assert.EqualValues(t, tc.msgResponse, *burnStableResponse)

			require.Equal(t, tc.supplyNIBI, nibiruApp.StablecoinKeeper.GetSupplyNIBI(ctx))
			require.Equal(t, tc.supplyNUSD, nibiruApp.StablecoinKeeper.GetSupplyNUSD(ctx))

			// Funds sypplies
			require.Equal(t,
				tc.ecosystemFund,
				nibiruApp.BankKeeper.GetAllBalances(
					ctx,
					nibiruApp.AccountKeeper.GetModuleAddress(types.StableEFModuleAccount)))
			require.Equal(t,
				tc.treasuryFund,
				nibiruApp.BankKeeper.GetAllBalances(
					ctx,
					nibiruApp.AccountKeeper.GetModuleAddress(common.TreasuryPoolModuleAccount)))
		})
	}
}

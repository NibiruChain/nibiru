package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	testutilapp "github.com/NibiruChain/nibiru/x/testutil/app"
	testutilevents "github.com/NibiruChain/nibiru/x/testutil/events"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestOpenPosition_Setup(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "open pos - uninitialized pool raised pair not supported error",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader without a vpool.")
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				pair, err := common.NewAssetPairFromStr("xxx:yyy")
				require.NoError(t, err)

				trader := sample.AccAddress()

				t.Log("open a position on invalid 'pair'")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				err = nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "open pos - vpool not set on the perp PairMetadata ",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				pair, err := common.NewAssetPairFromStr("xxx:yyy")
				require.NoError(t, err)

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					sdk.NewDec(10_000_000),       //
					sdk.NewDec(5_000_000),        // 5 tokens
					sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
					sdk.MustNewDecFromStr("0.1"),
				)

				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Attempt to open long position (expected unsuccessful)")
				trader := sample.AccAddress()
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				err = nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)

				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairMetadataNotFound.Error())
			},
		},
		{
			name: "open pos - happy path 1",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				pair, err := common.NewAssetPairFromStr("xxx:yyy")
				require.NoError(t, err)

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					sdk.NewDec(10_000_000),       //
					sdk.NewDec(5_000_000),        // 5 tokens
					sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
					sdk.MustNewDecFromStr("0.1"),
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.OneDec()},
				})

				t.Log("Fund trader account with sufficient quote")

				trader := sample.AccAddress()
				err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, trader,
					sdk.NewCoins(sdk.NewInt64Coin("yyy", 60)))
				require.NoError(t, err)

				t.Log("Open long position with 10x leverage")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				err = nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)

				require.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestAddMargin_ShouldRaiseError(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "msg denom differs from pair quote asset",
			test: func() {
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)

				tokenPair, err := common.NewAssetPairFromStr("atom:nusd")
				require.NoError(t, err)

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					/* pair */ tokenPair,
					/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"),
					/* quoteReserves */ sdk.NewDec(10_000_000), //
					/* baseReserves */ sdk.NewDec(5_000_000), // 5 tokens
					/* fluctuationLimitRatio */ sdk.MustNewDecFromStr("0.1"),
					/* maxOracleSpreadRatio */ sdk.OneDec(),
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, tokenPair))

				t.Log("create msg for MsgAddMargin with invalid denom")
				traderAddr := sample.AccAddress()
				msg := &types.MsgAddMargin{
					Sender:    traderAddr.String(),
					TokenPair: tokenPair.String(),
					Margin:    sdk.NewCoin("notADenom", sdk.NewInt(400)),
				}

				goCtx := sdk.WrapSDKContext(ctx)
				_, err = nibiruApp.PerpKeeper.AddMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "invalid margin denom")
			},
		},
	}

	for _, testCase := range tests {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestAddMargin_HappyPath(t *testing.T) {
	tests := []struct {
		name           string
		initialMargin  sdk.Int
		addedMargin    sdk.Int
		expectedMargin sdk.Int
	}{
		{
			name:           "add margin",
			initialMargin:  sdk.NewInt(100),
			addedMargin:    sdk.NewInt(100),
			expectedMargin: sdk.NewInt(200),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testutilapp.NewNibiruApp(true)

			tokenPair, err := common.NewAssetPairFromStr("atom:nusd")
			require.NoError(t, err)

			t.Log("add margin funds (NUSD) to trader's account")
			traderAddr := sample.AccAddress()
			err = simapp.FundAccount(
				nibiruApp.BankKeeper,
				ctx,
				traderAddr,
				sdk.NewCoins(
					sdk.NewCoin(tokenPair.GetQuoteTokenDenom(), tc.addedMargin),
				),
			)
			require.NoErrorf(t, err, "fund account call should work")

			t.Log("Set vpool defined by pair on VpoolKeeper")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				tokenPair,
				sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdk.NewDec(10_000_000),       //
				sdk.NewDec(5_000_000),        // 5 tokens
				sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
				/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("1.0"), // 100%
			)
			require.True(t, vpoolKeeper.ExistsPool(ctx, tokenPair))

			t.Log("Set vpool defined by pair on PerpKeeper")
			perpKeeper := &nibiruApp.PerpKeeper
			perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       tokenPair.String(),
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			t.Log("establish initial position")
			nibiruApp.PerpKeeper.SetPosition(
				ctx,
				tokenPair,
				traderAddr,
				&types.Position{
					TraderAddress: traderAddr.String(),
					Pair:          tokenPair.String(),
					Size_:         sdk.NewDec(9999),
					Margin:        tc.initialMargin.ToDec(),
				},
			)

			goCtx := sdk.WrapSDKContext(ctx)
			msg := &types.MsgAddMargin{
				Sender: traderAddr.String(), TokenPair: tokenPair.String(),
				Margin: sdk.Coin{
					Denom:  tokenPair.GetQuoteTokenDenom(),
					Amount: tc.addedMargin}}
			_, err = nibiruApp.PerpKeeper.AddMargin(goCtx, msg)
			require.NoError(t, err)

			position, err := nibiruApp.PerpKeeper.GetPosition(
				ctx, tokenPair, traderAddr)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedMargin, position.Margin.TruncateInt())
		})
	}
}

func TestRemoveMargin(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "negative margin remove - fail",
			test: func() {
				removeAmt := sdk.NewInt(-5)

				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				trader := sample.AccAddress()
				pair, err := common.NewAssetPairFromStr("osmo:nusd")
				require.NoError(t, err)

				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: trader.String(), TokenPair: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				_, err = nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "margin must be positive")
			},
		},
		{
			name: "zero margin remove - fail",
			test: func() {
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				trader := sample.AccAddress()
				pair, err := common.NewAssetPairFromStr("osmo:nusd")
				require.NoError(t, err)

				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender:    trader.String(),
					TokenPair: pair.String(),
					Margin: sdk.Coin{
						Denom:  common.StableDenom,
						Amount: sdk.ZeroInt(),
					},
				}
				_, err = nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "margin must be positive")
			},
		},
		{
			name: "vpool doesn't exit - fail",
			test: func() {
				removeAmt := sdk.NewInt(5)

				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				trader := sample.AccAddress()
				pair, err := common.NewAssetPairFromStr("osmo:nusd")
				require.NoError(t, err)

				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: trader.String(), TokenPair: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				_, err = nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "pool exists but trader doesn't have position - fail",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				trader := sample.AccAddress()
				pair, err := common.NewAssetPairFromStr("osmo:nusd")
				require.NoError(t, err)

				t.Log("Setup vpool defined by pair")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				perpKeeper := &nibiruApp.PerpKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ sdk.NewDec(1_000_000), //
					/* x */ sdk.NewDec(1_000_000), //
					/* fluctuationLimit */ sdk.MustNewDecFromStr("1.0"), // 100%
					/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("1.0"), // 100%
				)

				removeAmt := sdk.NewInt(5)
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: trader.String(), TokenPair: pair.String(),
					Margin: sdk.Coin{Denom: pair.GetQuoteTokenDenom(), Amount: removeAmt}}
				_, err = perpKeeper.RemoveMargin(
					goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
			},
		},
		{
			name: "remove margin from healthy position - fast integration test 1",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				traderAddr := sample.AccAddress()
				pair, err := common.NewAssetPairFromStr("xxx:yyy")
				require.NoError(t, err)

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				quoteReserves := sdk.NewDec(1_000_000)
				baseReserves := sdk.NewDec(1_000_000)
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ quoteReserves,
					/* x */ baseReserves,
					/* fluctuationLimit */ sdk.MustNewDecFromStr("1.0"), // 0.9 ratio
					/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("0.4"), // 0.9 ratio
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.1")},
				})

				t.Log("increment block height and time for twap calculation")
				ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
					WithBlockTime(time.Now().Add(time.Minute))

				t.Log("Fund trader account with sufficient quote")

				err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr,
					sdk.NewCoins(
						sdk.NewInt64Coin("yyy", 66),
					))
				require.NoError(t, err)

				t.Log("Open long position with 5x leverage")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(5)
				baseLimit := sdk.NewInt(10)
				err = nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, traderAddr, quote, leverage, baseLimit.ToDec())
				require.NoError(t, err)

				t.Log("Position should be accessible following 'OpenPosition'")
				_, err = nibiruApp.PerpKeeper.GetPosition(ctx, pair, traderAddr)
				require.NoError(t, err)

				t.Log("Verify correct events emitted for 'OpenPosition'")

				t.Log("Attempt to remove 10% of the position")
				removeAmt := sdk.NewInt(6)
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: traderAddr.String(), TokenPair: pair.String(),
					Margin: sdk.Coin{Denom: pair.GetQuoteTokenDenom(), Amount: removeAmt}}

				t.Log("'RemoveMargin' from the position")
				res, err := perpKeeper.RemoveMargin(goCtx, msg)
				require.NoError(t, err)
				assert.EqualValues(t, msg.Margin, res.MarginOut)
				assert.EqualValues(t, sdk.ZeroDec(), res.FundingPayment)

				t.Log("Verify correct events emitted for 'RemoveMargin'")
				testutilevents.RequireHasTypedEvent(t, ctx, &types.MarginChangedEvent{
					Pair:           pair.String(),
					TraderAddress:  traderAddr,
					MarginAmount:   msg.Margin.Amount,
					FundingPayment: res.FundingPayment,
				})
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

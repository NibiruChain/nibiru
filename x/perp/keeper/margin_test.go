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
	testutilevents "github.com/NibiruChain/nibiru/x/testutil/events"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
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
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				pair := common.MustNewAssetPair("xxx:yyy")

				trader := sample.AccAddress()

				t.Log("open a position on invalid 'pair'")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "open pos - vpool not set on the perp PairMetadata ",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				pair := common.MustNewAssetPair("xxx:yyy")

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
				err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)

				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairMetadataNotFound.Error())
			},
		},
		{
			name: "open pos - happy path 1",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				pair := common.MustNewAssetPair("xxx:yyy")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* quoteAssetReserve */ sdk.NewDec(10_000_000), //
					/* baseAssetReserve */ sdk.NewDec(5_000_000), // 5 tokens
					/* fluctuationLimitRatio */ sdk.MustNewDecFromStr("0.1"), // 0.1 ratio
					/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("0.1"),
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
					Pair: pair,
					CumulativePremiumFractions: []sdk.Dec{
						sdk.OneDec()},
				})

				t.Log("Fund trader account with sufficient quote")

				trader := sample.AccAddress()
				err := simapp.FundAccount(nibiruApp.BankKeeper, ctx, trader,
					sdk.NewCoins(sdk.NewInt64Coin("yyy", 62))) // extra 2yyy for fees
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

func TestAddMarginError(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "msg denom differs from pair quote asset",
			test: func() {
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					/* pair */ common.PairBTCStable,
					/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"),
					/* quoteReserves */ sdk.NewDec(10_000_000), //
					/* baseReserves */ sdk.NewDec(5_000_000), // 5 tokens
					/* fluctuationLimitRatio */ sdk.MustNewDecFromStr("0.1"),
					/* maxOracleSpreadRatio */ sdk.OneDec(),
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, common.PairBTCStable))

				t.Log("create msg for MsgAddMargin with invalid denom")
				msg := &types.MsgAddMargin{
					Sender:    sample.AccAddress().String(),
					TokenPair: common.PairBTCStable.String(),
					Margin:    sdk.NewCoin("notADenom", sdk.NewInt(400)),
				}

				_, err := nibiruApp.PerpKeeper.AddMargin(sdk.WrapSDKContext(ctx), msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "invalid margin denom")
			},
		},
		{
			name: "invalid sender",
			test: func() {
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)

				t.Log("create vpool")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					/* pair */ common.PairBTCStable,
					/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"),
					/* quoteReserves */ sdk.NewDec(10_000_000), //
					/* baseReserves */ sdk.NewDec(5_000_000), // 5 tokens
					/* fluctuationLimitRatio */ sdk.MustNewDecFromStr("0.1"),
					/* maxOracleSpreadRatio */ sdk.OneDec(),
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, common.PairBTCStable))

				t.Log("create msg for MsgAddMargin with invalid denom")
				msg := &types.MsgAddMargin{
					Sender:    "",
					TokenPair: common.PairBTCStable.String(),
					Margin:    sdk.NewCoin("unusd", sdk.NewInt(400)),
				}

				_, err := nibiruApp.PerpKeeper.AddMargin(sdk.WrapSDKContext(ctx), msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "empty address string is not allowed")
			},
		},
		{
			name: "invalid negative margin add",
			test: func() {
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)

				t.Log("create vpool")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					/* pair */ common.PairBTCStable,
					/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"),
					/* quoteReserves */ sdk.NewDec(10_000_000), //
					/* baseReserves */ sdk.NewDec(5_000_000), // 5 tokens
					/* fluctuationLimitRatio */ sdk.MustNewDecFromStr("0.1"),
					/* maxOracleSpreadRatio */ sdk.OneDec(),
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, common.PairBTCStable))

				t.Log("create msg for MsgAddMargin with invalid denom")
				msg := &types.MsgAddMargin{
					Sender:    sample.AccAddress().String(),
					TokenPair: common.PairBTCStable.String(),
					Margin:    sdk.Coin{Denom: "unusd", Amount: sdk.NewInt(-400)},
				}

				_, err := nibiruApp.PerpKeeper.AddMargin(sdk.WrapSDKContext(ctx), msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "margin must be positive")
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

func TestAddMarginSuccess(t *testing.T) {
	tests := []struct {
		name                            string
		marginToAdd                     sdk.Int
		latestCumulativePremiumFraction sdk.Dec
		initialPosition                 types.Position

		expectedMargin         sdk.Dec
		expectedFundingPayment sdk.Dec
	}{
		{
			name:                            "add margin",
			marginToAdd:                     sdk.NewInt(100),
			latestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.001"),
			initialPosition: types.Position{
				TraderAddress:                       sample.AccAddress().String(),
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(1_000),
				Margin:                              sdk.NewDec(100),
				OpenNotional:                        sdk.NewDec(500),
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         1,
			},

			expectedMargin:         sdk.NewDec(199),
			expectedFundingPayment: sdk.NewDec(1),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)

			traderAddr, err := sdk.AccAddressFromBech32(tc.initialPosition.TraderAddress)
			require.NoError(t, err)

			t.Log("add trader funds")
			err = simapp.FundAccount(
				nibiruApp.BankKeeper,
				ctx,
				traderAddr,
				sdk.NewCoins(
					sdk.NewCoin(common.PairBTCStable.GetQuoteTokenDenom(), tc.marginToAdd),
				),
			)
			require.NoErrorf(t, err, "fund account call should work")

			t.Log("create vpool")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				common.PairBTCStable,
				sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdk.NewDec(10_000_000),       //
				sdk.NewDec(5_000_000),        // 5 tokens
				sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
				/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("1.0"), // 100%
			)
			require.True(t, vpoolKeeper.ExistsPool(ctx, common.PairBTCStable))

			t.Log("set pair metadata")
			perpKeeper := &nibiruApp.PerpKeeper
			perpKeeper.PairMetadataState(ctx).Set(
				&types.PairMetadata{
					Pair: common.PairBTCStable,
					CumulativePremiumFractions: []sdk.Dec{
						tc.latestCumulativePremiumFraction,
					},
				},
			)

			t.Log("establish initial position")
			nibiruApp.PerpKeeper.SetPosition(
				ctx,
				common.PairBTCStable,
				traderAddr,
				&tc.initialPosition,
			)

			goCtx := sdk.WrapSDKContext(ctx)
			msg := &types.MsgAddMargin{
				Sender:    traderAddr.String(),
				TokenPair: tc.initialPosition.Pair.String(),
				Margin: sdk.Coin{
					Denom:  common.PairBTCStable.GetQuoteTokenDenom(),
					Amount: tc.marginToAdd,
				},
			}
			resp, err := nibiruApp.PerpKeeper.AddMargin(goCtx, msg)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedFundingPayment, resp.FundingPayment)
			assert.EqualValues(t, tc.expectedMargin, resp.Position.Margin)
			assert.EqualValues(t, tc.initialPosition.OpenNotional, resp.Position.OpenNotional)
			assert.EqualValues(t, tc.initialPosition.Size_, resp.Position.Size_)
			assert.EqualValues(t, traderAddr.String(), resp.Position.TraderAddress)
			assert.EqualValues(t, common.PairBTCStable, resp.Position.Pair)
			assert.EqualValues(t, tc.latestCumulativePremiumFraction, resp.Position.LastUpdateCumulativePremiumFraction)
			assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
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

				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				trader := sample.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender:    trader.String(),
					TokenPair: pair.String(),
					Margin:    sdk.Coin{Denom: common.DenomStable, Amount: removeAmt},
				}
				_, err := nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "margin must be positive")
			},
		},
		{
			name: "zero margin remove - fail",
			test: func() {
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				trader := sample.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender:    trader.String(),
					TokenPair: pair.String(),
					Margin: sdk.Coin{
						Denom:  common.DenomStable,
						Amount: sdk.ZeroInt(),
					},
				}
				_, err := nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "margin must be positive")
			},
		},
		{
			name: "vpool doesn't exit - fail",
			test: func() {
				removeAmt := sdk.NewInt(5)

				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				trader := sample.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender:    trader.String(),
					TokenPair: pair.String(),
					Margin:    sdk.Coin{Denom: common.DenomStable, Amount: removeAmt},
				}
				_, err := nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "pool exists but trader doesn't have position - fail",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				trader := sample.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

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
					Sender:    trader.String(),
					TokenPair: pair.String(),
					Margin:    sdk.Coin{Denom: pair.GetQuoteTokenDenom(), Amount: removeAmt},
				}
				_, err := perpKeeper.RemoveMargin(
					goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
			},
		},
		{
			name: "remove margin from healthy position - fast integration test 1",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				traderAddr := sample.AccAddress()
				pair := common.MustNewAssetPair("xxx:yyy")

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
				perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
					Pair: pair,
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.1")},
				})

				t.Log("increment block height and time for twap calculation")
				ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
					WithBlockTime(time.Now().Add(time.Minute))

				t.Log("Fund trader account with sufficient quote")

				err := simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr,
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
					Sender:    traderAddr.String(),
					TokenPair: pair.String(),
					Margin:    sdk.Coin{Denom: pair.GetQuoteTokenDenom(), Amount: removeAmt},
				}

				t.Log("'RemoveMargin' from the position")
				res, err := perpKeeper.RemoveMargin(goCtx, msg)
				require.NoError(t, err)
				assert.EqualValues(t, msg.Margin, res.MarginOut)
				assert.EqualValues(t, sdk.ZeroDec(), res.FundingPayment)

				t.Log("Verify correct events emitted for 'RemoveMargin'")
				testutilevents.RequireHasTypedEvent(t, ctx, &types.MarginChangedEvent{
					Pair:           pair.String(),
					TraderAddress:  traderAddr.String(),
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

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

func TestAddMarginSuccess(t *testing.T) {
	tests := []struct {
		name                            string
		marginToAdd                     sdk.Coin
		latestCumulativePremiumFraction sdk.Dec
		initialPosition                 types.Position

		expectedMargin         sdk.Dec
		expectedFundingPayment sdk.Dec
	}{
		{
			name:                            "add margin",
			marginToAdd:                     sdk.NewInt64Coin(common.DenomStable, 100),
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
			traderAddr := sdk.MustAccAddressFromBech32(tc.initialPosition.TraderAddress)

			t.Log("add trader funds")
			require.NoError(t, simapp.FundAccount(
				nibiruApp.BankKeeper,
				ctx,
				traderAddr,
				sdk.NewCoins(tc.marginToAdd),
			))

			t.Log("create vpool")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				common.PairBTCStable,
				/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdk.NewDec(10_000_000),                             // 10 tokens
				sdk.NewDec(5_000_000),                              // 5 tokens
				/* fluctuationLimitRatio */ sdk.MustNewDecFromStr("0.1"), // 0.1 ratio
				/* maxOracleSpreadRatio */ sdk.OneDec(), // 100%
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
			nibiruApp.PerpKeeper.PositionsState(ctx).Set(&tc.initialPosition)

			resp, err := nibiruApp.PerpKeeper.AddMargin(ctx, common.PairBTCStable, traderAddr, tc.marginToAdd)
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
			name: "vpool doesn't exit - fail",
			test: func() {
				removeAmt := sdk.NewInt(5)

				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				trader := sample.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				_, _, _, err := nibiruApp.PerpKeeper.RemoveMargin(ctx, pair, trader, sdk.Coin{Denom: common.DenomStable, Amount: removeAmt})
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
				_, _, _, err := perpKeeper.RemoveMargin(ctx, pair, trader, sdk.Coin{Denom: pair.GetQuoteTokenDenom(), Amount: removeAmt})

				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
			},
		},
		{
			name: "remove margin from healthy position",
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
					Pair:                       pair,
					CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
				})

				t.Log("increment block height and time for twap calculation")
				ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
					WithBlockTime(time.Now().Add(time.Minute))

				t.Log("Fund trader account with sufficient quote")
				require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr,
					sdk.NewCoins(sdk.NewInt64Coin("yyy", 60))),
				)

				t.Log("Open long position with 5x leverage")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(5)
				baseLimit := sdk.ZeroDec()
				_, err := perpKeeper.OpenPosition(ctx, pair, side, traderAddr, quote, leverage, baseLimit)
				require.NoError(t, err)

				t.Log("Attempt to remove 10% of the position")
				removeAmt := sdk.NewInt(6)

				t.Log("'RemoveMargin' from the position")
				marginOut, fundingPayment, position, err := perpKeeper.RemoveMargin(ctx, pair, traderAddr, sdk.Coin{Denom: pair.GetQuoteTokenDenom(), Amount: removeAmt})
				require.NoError(t, err)
				assert.EqualValues(t, sdk.Coin{Denom: pair.GetQuoteTokenDenom(), Amount: removeAmt}, marginOut)
				assert.EqualValues(t, sdk.ZeroDec(), fundingPayment)
				assert.EqualValues(t, pair, position.Pair)
				assert.EqualValues(t, traderAddr.String(), position.TraderAddress)
				assert.EqualValues(t, sdk.NewDec(54), position.Margin)
				assert.EqualValues(t, sdk.NewDec(300), position.OpenNotional)
				assert.EqualValues(t, sdk.MustNewDecFromStr("299.910026991902429271"), position.Size_)
				assert.EqualValues(t, ctx.BlockHeight(), ctx.BlockHeight())
				assert.EqualValues(t, sdk.ZeroDec(), position.LastUpdateCumulativePremiumFraction)

				t.Log("Verify correct events emitted for 'RemoveMargin'")
				testutilevents.RequireContainsTypedEvent(t, ctx,
					&types.PositionChangedEvent{
						Pair:                  pair.String(),
						TraderAddress:         traderAddr.String(),
						Margin:                sdk.NewInt64Coin(pair.GetQuoteTokenDenom(), 54),
						PositionNotional:      sdk.NewDec(300),
						ExchangedPositionSize: sdk.ZeroDec(),                                         // always zero when removing margin
						TransactionFee:        sdk.NewCoin(pair.GetQuoteTokenDenom(), sdk.ZeroInt()), // always zero when removing margin
						PositionSize:          sdk.MustNewDecFromStr("299.910026991902429271"),
						RealizedPnl:           sdk.ZeroDec(), // always zero when removing margin
						UnrealizedPnlAfter:    sdk.ZeroDec(),
						BadDebt:               sdk.NewCoin(pair.GetQuoteTokenDenom(), sdk.ZeroInt()), // always zero when adding margin
						FundingPayment:        sdk.ZeroDec(),
						SpotPrice:             sdk.MustNewDecFromStr("1.00060009"),
						BlockHeight:           ctx.BlockHeight(),
						BlockTimeMs:           ctx.BlockTime().UnixMilli(),
						LiquidationPenalty:    sdk.ZeroDec(),
					},
				)
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

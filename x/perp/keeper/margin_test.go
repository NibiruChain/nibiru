package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"

	testutilevents "github.com/NibiruChain/nibiru/x/testutil"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"

	simapp2 "github.com/NibiruChain/nibiru/simapp"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
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
			marginToAdd:                     sdk.NewInt64Coin(common.DenomNUSD, 100),
			latestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.001"),
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.Pair_BTC_NUSD,
				Size_:                           sdk.NewDec(1_000),
				Margin:                          sdk.NewDec(100),
				OpenNotional:                    sdk.NewDec(500),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},

			expectedMargin:         sdk.NewDec(199),
			expectedFundingPayment: sdk.NewDec(1),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
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
				common.Pair_BTC_NUSD,
				sdk.NewDec(10*common.Precision), // 10 tokens
				sdk.NewDec(5*common.Precision),  // 5 tokens
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"), // 0.1 ratio
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			)
			require.True(t, vpoolKeeper.ExistsPool(ctx, common.Pair_BTC_NUSD))

			t.Log("set pair metadata")
			setPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            common.Pair_BTC_NUSD,
				LatestCumulativePremiumFraction: tc.latestCumulativePremiumFraction,
			},
			)

			t.Log("establish initial position")
			setPosition(nibiruApp.PerpKeeper, ctx, tc.initialPosition)

			resp, err := nibiruApp.PerpKeeper.AddMargin(ctx, common.Pair_BTC_NUSD, traderAddr, tc.marginToAdd)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedFundingPayment, resp.FundingPayment)
			assert.EqualValues(t, tc.expectedMargin, resp.Position.Margin)
			assert.EqualValues(t, tc.initialPosition.OpenNotional, resp.Position.OpenNotional)
			assert.EqualValues(t, tc.initialPosition.Size_, resp.Position.Size_)
			assert.EqualValues(t, traderAddr.String(), resp.Position.TraderAddress)
			assert.EqualValues(t, common.Pair_BTC_NUSD, resp.Position.Pair)
			assert.EqualValues(t, tc.latestCumulativePremiumFraction, resp.Position.LatestCumulativePremiumFraction)
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

				nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
				trader := testutilevents.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				_, _, _, err := nibiruApp.PerpKeeper.RemoveMargin(ctx, pair, trader, sdk.Coin{Denom: common.DenomNUSD, Amount: removeAmt})
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "pool exists but trader doesn't have position - fail",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
				trader := testutilevents.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				t.Log("Setup vpool defined by pair")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				perpKeeper := &nibiruApp.PerpKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					/* y */ sdk.NewDec(1*common.Precision), //
					/* x */ sdk.NewDec(1*common.Precision), //
					vpooltypes.VpoolConfig{
						TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
						FluctuationLimitRatio:  sdk.OneDec(),
						MaxOracleSpreadRatio:   sdk.OneDec(),
						MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
						MaxLeverage:            sdk.MustNewDecFromStr("15"),
					},
				)

				removeAmt := sdk.NewInt(5)
				_, _, _, err := perpKeeper.RemoveMargin(ctx, pair, trader, sdk.Coin{Denom: pair.QuoteDenom(), Amount: removeAmt})

				require.Error(t, err)
				require.ErrorContains(t, err, collections.ErrNotFound.Error())
			},
		},
		{
			name: "remove margin from healthy position",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
				ctx = ctx.WithBlockTime(time.Now())
				traderAddr := testutilevents.AccAddress()
				pair := common.MustNewAssetPair("xxx:yyy")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				quoteReserves := sdk.NewDec(1 * common.Precision)
				baseReserves := sdk.NewDec(1 * common.Precision)
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					/* y */ quoteReserves,
					/* x */ baseReserves,
					vpooltypes.VpoolConfig{
						TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
						FluctuationLimitRatio:  sdk.OneDec(),
						MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.4"),
						MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
						MaxLeverage:            sdk.MustNewDecFromStr("15"),
					},
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				setPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})

				t.Log("increment block height and time for twap calculation")
				ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
					WithBlockTime(time.Now().Add(time.Minute))

				t.Log("Fund trader account with sufficient quote")
				require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr,
					sdk.NewCoins(sdk.NewInt64Coin("yyy", 60))),
				)

				t.Log("Open long position with 5x leverage")
				perpKeeper := nibiruApp.PerpKeeper
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(5)
				baseLimit := sdk.ZeroDec()
				_, err := perpKeeper.OpenPosition(ctx, pair, side, traderAddr, quote, leverage, baseLimit)
				require.NoError(t, err)

				t.Log("Attempt to remove 10% of the position")
				removeAmt := sdk.NewInt(6)

				t.Log("'RemoveMargin' from the position")
				marginOut, fundingPayment, position, err := perpKeeper.RemoveMargin(ctx, pair, traderAddr, sdk.Coin{Denom: pair.QuoteDenom(), Amount: removeAmt})
				require.NoError(t, err)
				assert.EqualValues(t, sdk.Coin{Denom: pair.QuoteDenom(), Amount: removeAmt}, marginOut)
				assert.EqualValues(t, sdk.ZeroDec(), fundingPayment)
				assert.EqualValues(t, pair, position.Pair)
				assert.EqualValues(t, traderAddr.String(), position.TraderAddress)
				assert.EqualValues(t, sdk.NewDec(54), position.Margin)
				assert.EqualValues(t, sdk.NewDec(300), position.OpenNotional)
				assert.EqualValues(t, sdk.MustNewDecFromStr("299.910026991902429271"), position.Size_)
				assert.EqualValues(t, ctx.BlockHeight(), ctx.BlockHeight())
				assert.EqualValues(t, sdk.ZeroDec(), position.LatestCumulativePremiumFraction)

				t.Log("Verify correct events emitted for 'RemoveMargin'")
				testutilevents.RequireContainsTypedEvent(t, ctx,
					&types.PositionChangedEvent{
						Pair:               pair.String(),
						TraderAddress:      traderAddr.String(),
						Margin:             sdk.NewInt64Coin(pair.QuoteDenom(), 54),
						PositionNotional:   sdk.NewDec(300),
						ExchangedNotional:  sdk.ZeroDec(),                                 // always zero when removing margin
						ExchangedSize:      sdk.ZeroDec(),                                 // always zero when removing margin
						TransactionFee:     sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when removing margin
						PositionSize:       sdk.MustNewDecFromStr("299.910026991902429271"),
						RealizedPnl:        sdk.ZeroDec(), // always zero when removing margin
						UnrealizedPnlAfter: sdk.ZeroDec(),
						BadDebt:            sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when adding margin
						FundingPayment:     sdk.ZeroDec(),
						MarkPrice:          sdk.MustNewDecFromStr("1.00060009"),
						BlockHeight:        ctx.BlockHeight(),
						BlockTimeMs:        ctx.BlockTime().UnixMilli(),
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

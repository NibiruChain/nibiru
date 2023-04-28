package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	testutilevents "github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	keeper "github.com/NibiruChain/nibiru/x/perp/keeper/v2"
	"github.com/NibiruChain/nibiru/x/perp/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestAddMarginSuccess(t *testing.T) {
	tests := []struct {
		name                            string
		marginToAdd                     sdk.Coin
		latestCumulativePremiumFraction sdk.Dec
		initialPosition                 v2types.Position

		expectedMargin         sdk.Dec
		expectedFundingPayment sdk.Dec
	}{
		{
			name:                            "add margin",
			marginToAdd:                     sdk.NewInt64Coin(denoms.NUSD, 100),
			latestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.001"),
			initialPosition: v2types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(1_000),
				Margin:                          sdk.NewDec(100),
				OpenNotional:                    sdk.NewDec(500),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			},

			expectedMargin:         sdk.NewDec(199),
			expectedFundingPayment: sdk.NewDec(1),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
			traderAddr := sdk.MustAccAddressFromBech32(tc.initialPosition.TraderAddress)

			t.Log("add trader funds")
			require.NoError(t, testapp.FundAccount(
				nibiruApp.BankKeeper,
				ctx,
				traderAddr,
				sdk.NewCoins(tc.marginToAdd),
			))

			t.Log("create market")
			nibiruApp.PerpKeeperV2.Markets.Insert(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), *mock.TestMarket())
			// assert.NoError(t, perpammKeeper.CreatePool(
			// 	ctx,
			// 	asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			// 	sdk.NewDec(5*common.TO_MICRO), // 10 tokens
			// 	sdk.NewDec(5*common.TO_MICRO), // 5 tokens
			// 	v2types.MarketConfig{
			// 		TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
			// 		FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"), // 0.1 ratio
			// 		MaxOracleSpreadRatio:   sdk.OneDec(),
			// 		MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			// 		MaxLeverage:            sdk.MustNewDecFromStr("15"),
			// 	},
			// 	sdk.NewDec(2),
			// ))

			t.Log("establish initial position")
			keeper.SetPosition(nibiruApp.PerpKeeperV2, ctx, tc.initialPosition)

			resp, err := nibiruApp.PerpKeeperV2.AddMargin(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr, tc.marginToAdd)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedFundingPayment, resp.FundingPayment)
			assert.EqualValues(t, tc.expectedMargin, resp.Position.Margin)
			assert.EqualValues(t, tc.initialPosition.OpenNotional, resp.Position.OpenNotional)
			assert.EqualValues(t, tc.initialPosition.Size_, resp.Position.Size_)
			assert.EqualValues(t, traderAddr.String(), resp.Position.TraderAddress)
			assert.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), resp.Position.Pair)
			assert.EqualValues(t, tc.latestCumulativePremiumFraction, resp.Position.LatestCumulativePremiumFraction)
			assert.EqualValues(t, ctx.BlockHeight(), resp.Position.LastUpdatedBlockNumber)
		})
	}
}

func TestRemoveMargin(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{

		{
			name: "market doesn't exit - fail",
			test: func() {
				removeAmt := sdk.NewInt(5)

				nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
				trader := testutilevents.AccAddress()
				pair := asset.MustNewPair("osmo:nusd")

				_, _, _, err := nibiruApp.PerpKeeperV2.RemoveMargin(ctx, pair, trader, sdk.Coin{Denom: denoms.NUSD, Amount: removeAmt})
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "pool exists but trader doesn't have position - fail",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
				trader := testutilevents.AccAddress()
				pair := asset.MustNewPair("osmo:nusd")

				t.Log("Setup market defined by pair")
				nibiruApp.PerpKeeperV2.Markets.Insert(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), *mock.TestMarket())

				// perpammKeeper := &nibiruApp.PerpAmmKeeper
				// perpKeeper := &nibiruApp.PerpKeeperV2
				// assert.NoError(t, perpammKeeper.CreatePool(
				// 	ctx,
				// 	pair,
				// 	/* y */ sdk.NewDec(1*common.TO_MICRO), //
				// 	/* x */ sdk.NewDec(1*common.TO_MICRO), //
				// 	v2types.MarketConfig{
				// 		TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
				// 		FluctuationLimitRatio:  sdk.OneDec(),
				// 		MaxOracleSpreadRatio:   sdk.OneDec(),
				// 		MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				// 		MaxLeverage:            sdk.MustNewDecFromStr("15"),
				// 	},
				// 	sdk.OneDec(),
				// ))

				removeAmt := sdk.NewInt(5)
				_, _, _, err := nibiruApp.PerpKeeperV2.RemoveMargin(ctx, pair, trader, sdk.Coin{Denom: pair.QuoteDenom(), Amount: removeAmt})

				require.Error(t, err)
				require.ErrorContains(t, err, collections.ErrNotFound.Error())
			},
		},
		{
			name: "remove margin from healthy position",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
				ctx = ctx.WithBlockTime(time.Now())
				traderAddr := testutilevents.AccAddress()
				pair := asset.MustNewPair("xxx:yyy")

				t.Log("Set market defined by pair on PerpAmmKeeper")
				// perpammKeeper := &nibiruApp.PerpAmmKeeper
				// quoteReserves := sdk.NewDec(1e6)
				// baseReserves := sdk.NewDec(1e6)
				// assert.NoError(t, perpammKeeper.CreatePool(
				// 	ctx,
				// 	pair,
				// 	/* y */ quoteReserves,
				// 	/* x */ baseReserves,
				// 	v2types.MarketConfig{
				// 		TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
				// 		FluctuationLimitRatio:  sdk.OneDec(),
				// 		MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.4"),
				// 		MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				// 		MaxLeverage:            sdk.MustNewDecFromStr("15"),
				// 	},
				// 	sdk.OneDec(),
				// ))
				nibiruApp.PerpKeeperV2.Markets.Insert(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), *mock.TestMarket())

				t.Log("increment block height and time for twap calculation")
				ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
					WithBlockTime(time.Now().Add(time.Minute))

				t.Log("Fund trader account with sufficient quote")
				require.NoError(t, testapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr,
					sdk.NewCoins(sdk.NewInt64Coin("yyy", 60))),
				)

				t.Log("Open long position with 5x leverage")
				perpKeeper := nibiruApp.PerpKeeperV2
				side := v2types.Direction_LONG
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
						Pair:               pair,
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

func TestGetSpotMarginRatio(t *testing.T) {
	tests := []struct {
		name                string
		position            v2types.Position
		positionNotional    sdk.Dec
		latestCPF           sdk.Dec
		expectedMarginRatio sdk.Dec
	}{
		{
			name: "long position, no change",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(100),
			latestCPF:           sdk.ZeroDec(),
			expectedMarginRatio: sdk.OneDec(),
		},
		{
			name: "long position, positive PnL, positive cumulative premium fraction",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(110),
			latestCPF:           sdk.NewDec(2),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.981818181818181818"), // 108 / 110
		},
		{
			name: "long position, positive PnL, negative cumulative premium fraction",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(110),
			latestCPF:           sdk.NewDec(-2),
			expectedMarginRatio: sdk.MustNewDecFromStr("1.018181818181818182"), // 112 / 110
		},
		{
			name: "long position, negative PnL, positive cumulative premium fraction",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(90),
			latestCPF:           sdk.NewDec(2),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.977777777777777778"), // 88 / 90
		},
		{
			name: "long position, negative PnL, negative cumulative premium fraction",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(90),
			latestCPF:           sdk.NewDec(-2),
			expectedMarginRatio: sdk.MustNewDecFromStr("1.022222222222222222"), // 92 / 90
		},
		{
			name: "short position, no change",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(100),
			latestCPF:           sdk.ZeroDec(),
			expectedMarginRatio: sdk.OneDec(),
		},
		{
			name: "short position, positive PnL, positive cumulative premium fraction",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(90),
			latestCPF:           sdk.NewDec(2),
			expectedMarginRatio: sdk.MustNewDecFromStr("1.244444444444444444"), // 112 / 90
		},
		{
			name: "short position, positive PnL, negative cumulative premium fraction",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(90),
			latestCPF:           sdk.NewDec(-2),
			expectedMarginRatio: sdk.MustNewDecFromStr("1.2"), // 108 / 90
		},
		{
			name: "short position, negative PnL, positive cumulative premium fraction",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(110),
			latestCPF:           sdk.NewDec(2),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.836363636363636364"), // 92 / 110
		},
		{
			name: "short position, negative PnL, negative cumulative premium fraction",
			position: v2types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(110),
			latestCPF:           sdk.NewDec(-2),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.8"), // 88 / 110
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			marginRatio, err := keeper.GetSpotMarginRatio(tc.position, tc.positionNotional, tc.latestCPF)

			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedMarginRatio, marginRatio)
		})
	}
}

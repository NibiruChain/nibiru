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
	"github.com/NibiruChain/nibiru/x/common/testutil"
	testutilevents "github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/oracle/integration_test/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion/v2"
	"github.com/NibiruChain/nibiru/x/perp/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestAddMargin(t *testing.T) {
	alice := testutil.AccAddress()
	PairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("existing long position, add margin").
			Given(
				createInitMarket(PairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetPairPrice(PairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2020)))),
				OpenPosition(alice, PairBtcUsdc, v2types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				AddMargin(alice, PairBtcUsdc, sdk.NewInt(1000)),
			).
			Then(
				PositionShouldBeEqual(alice, PairBtcUsdc, Position_PositionShouldBeEqualTo(v2types.Position{
					Pair:                            PairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("9999.999900000001000000"),
					Margin:                          sdk.NewDec(2000),
					OpenNotional:                    sdk.NewDec(10000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&v2types.PositionChangedEvent{
					Pair:               PairBtcUsdc,
					TraderAddress:      alice.String(),
					Margin:             sdk.NewCoin(denoms.USDC, sdk.NewInt(2000)),
					PositionNotional:   sdk.NewDec(10_000),
					ExchangedNotional:  sdk.ZeroDec(),
					ExchangedSize:      sdk.ZeroDec(),
					PositionSize:       sdk.MustNewDecFromStr("9999.999900000001000000"),
					RealizedPnl:        sdk.ZeroDec(),
					UnrealizedPnlAfter: sdk.ZeroDec(),
					BadDebt:            sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					FundingPayment:     sdk.ZeroDec(),
					TransactionFee:     sdk.NewCoin(denoms.USDC, sdk.NewInt(20)), // 20 bps
					BlockHeight:        2,
					BlockTimeMs:        startBlockTime.Add(time.Second*5).UnixNano() / 1e6,
				}),
			),

		TC("existing short position, add margin").
			Given(
				createInitMarket(PairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetPairPrice(PairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2020)))),
				OpenPosition(alice, PairBtcUsdc, v2types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				AddMargin(alice, PairBtcUsdc, sdk.NewInt(1000)),
			).
			Then(
				PositionShouldBeEqual(alice, PairBtcUsdc, Position_PositionShouldBeEqualTo(v2types.Position{
					Pair:                            PairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
					Margin:                          sdk.NewDec(2000),
					OpenNotional:                    sdk.NewDec(10000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&v2types.PositionChangedEvent{
					Pair:               PairBtcUsdc,
					TraderAddress:      alice.String(),
					Margin:             sdk.NewCoin(denoms.USDC, sdk.NewInt(2000)),
					PositionNotional:   sdk.NewDec(10_000),
					ExchangedNotional:  sdk.ZeroDec(),
					ExchangedSize:      sdk.ZeroDec(),
					PositionSize:       sdk.MustNewDecFromStr("-10000.000100000001000000"),
					RealizedPnl:        sdk.ZeroDec(),
					UnrealizedPnlAfter: sdk.ZeroDec(),
					BadDebt:            sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					FundingPayment:     sdk.ZeroDec(),
					TransactionFee:     sdk.NewCoin(denoms.USDC, sdk.NewInt(20)), // 20 bps
					BlockHeight:        2,
					BlockTimeMs:        startBlockTime.Add(time.Second*5).UnixNano() / 1e6,
				}),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
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

				_, _, _, err := nibiruApp.PerpKeeperV2.RemoveMargin(ctx, pair, trader, sdk.Coin{Denom: denoms.USDC, Amount: removeAmt})
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
				nibiruApp.PerpKeeperV2.Markets.Insert(ctx, asset.Registry.Pair(denoms.BTC, denoms.USDC), *mock.TestMarket())

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
				nibiruApp.PerpKeeperV2.Markets.Insert(ctx, asset.Registry.Pair(denoms.BTC, denoms.USDC), *mock.TestMarket())

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

package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/oracle/integration_test/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion/v2"
	"github.com/NibiruChain/nibiru/x/perp/types"

	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func createInitMarket() Action {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)

	return CreateCustomMarket(
		v2types.Market{
			Pair:                            pairBtcUsdc,
			Enabled:                         true,
			LatestCumulativePremiumFraction: sdk.ZeroDec(),
			ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.001"),
			EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.001"),
			LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.005"),
			PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
			FundingRateEpochId:              "30min",
			TwapLookbackWindow:              time.Minute * 30,
			WhitelistedLiquidators:          []string{},
			PrepaidBadDebt:                  sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
			PriceFluctuationLimitRatio:      sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:                     sdk.NewDec(10),
			MaxOracleSpreadRatio:            sdk.OneDec(), // 100%,
		},
		v2types.AMM{
			Pair:            pairBtcUsdc,
			BaseReserve:     sdk.NewDec(1e12),
			QuoteReserve:    sdk.NewDec(1e12),
			SqrtDepth:       sdk.NewDec(1e12),
			PriceMultiplier: sdk.OneDec(),
			Bias:            sdk.ZeroDec(),
		})
}

func TestOpenPosition(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("new long position").
			Given(
				createInitMarket(),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetPairPrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1020)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, v2types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						v2types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(10_000),
							Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
							LastUpdatedBlockNumber:          1,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(1000*10)), // margin * leverage
					OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9999.999900000001")),
					OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					OpenPositionResp_PositionNotionalShouldBeEqual(sdk.NewDec(10_000)),
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(v2types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.NewDec(1000),
					OpenNotional:                    sdk.NewDec(10_000),
					Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})),
				PositionChangedEventShouldBeEqual(&v2types.PositionChangedEvent{
					Pair:               pairBtcUsdc,
					TraderAddress:      alice.String(),
					Margin:             sdk.NewCoin(denoms.USDC, sdk.NewDec(1000).TruncateInt()),
					PositionNotional:   sdk.NewDec(10_000),
					ExchangedNotional:  sdk.NewDec(1000 * 10),
					ExchangedSize:      sdk.MustNewDecFromStr("9999.999900000001"),
					PositionSize:       sdk.MustNewDecFromStr("9999.999900000001"),
					RealizedPnl:        sdk.ZeroDec(),
					UnrealizedPnlAfter: sdk.ZeroDec(),
					BadDebt:            sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					FundingPayment:     sdk.ZeroDec(),
					TransactionFee:     sdk.NewCoin(denoms.USDC, sdk.NewInt(20)),
					BlockHeight:        1,
					BlockTimeMs:        startBlockTime.UnixNano() / 1e6,
				}),
			),

		TC("existing long position, go more long").Given(
			createInitMarket(),
			SetBlockNumber(1),
			SetBlockTime(startBlockTime),
			SetPairPrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2040)))),
			OpenPosition(alice, pairBtcUsdc, v2types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
		).When(
			MoveToNextBlock(),
			OpenPosition(alice, pairBtcUsdc, v2types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
				OpenPositionResp_PositionShouldBeEqual(
					v2types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(2000),
						OpenNotional:                    sdk.NewDec(20_000),
						Size_:                           sdk.MustNewDecFromStr("19999.999600000008000000"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					}),
				OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(1000*10)), // margin * leverage
				OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9999.999700000007000000")),
				OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
				OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
				OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
				OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.MustNewDecFromStr("0.000199999998000000")),
				OpenPositionResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
				OpenPositionResp_PositionNotionalShouldBeEqual(sdk.MustNewDecFromStr("20000.000199999998000000")),
			),
		).Then(
			PositionChangedEventShouldBeEqual(&v2types.PositionChangedEvent{
				Pair:               pairBtcUsdc,
				TraderAddress:      alice.String(),
				Margin:             sdk.NewCoin(denoms.USDC, sdk.NewDec(2000).TruncateInt()),
				PositionNotional:   sdk.MustNewDecFromStr("20000.000199999998000000"),
				ExchangedNotional:  sdk.NewDec(1000 * 10),
				ExchangedSize:      sdk.MustNewDecFromStr("9999.999700000007000000"),
				PositionSize:       sdk.MustNewDecFromStr("19999.999600000008000000"),
				RealizedPnl:        sdk.ZeroDec(),
				UnrealizedPnlAfter: sdk.MustNewDecFromStr("0.000199999998000000"),
				BadDebt:            sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
				FundingPayment:     sdk.ZeroDec(),
				TransactionFee:     sdk.NewCoin(denoms.USDC, sdk.NewInt(20)),
				BlockHeight:        2,
				BlockTimeMs:        startBlockTime.Add(time.Second*5).UnixNano() / 1e6,
			}),
		),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestOpenPositionSuccess(t *testing.T) {
	testCases := []struct {
		name        string
		traderFunds sdk.Coins

		initialPosition *v2types.Position

		side      v2types.Direction
		margin    sdk.Int
		leverage  sdk.Dec
		baseLimit sdk.Dec

		expectedMargin           sdk.Dec
		expectedOpenNotional     sdk.Dec
		expectedSize             sdk.Dec
		expectedPositionNotional sdk.Dec
		expectedUnrealizedPnl    sdk.Dec
		expectedRealizedPnl      sdk.Dec
		expectedMarginToVault    sdk.Dec
		expectedMarkPrice        sdk.Dec
	}{
		{
			name:                     "new long position",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition:          nil,
			side:                     v2types.Direction_LONG,
			margin:                   sdk.NewInt(1000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(1000),
			expectedOpenNotional:     sdk.NewDec(10_000),
			expectedSize:             sdk.MustNewDecFromStr("9999.999900000001"),
			expectedPositionNotional: sdk.NewDec(10_000),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.0000000200000001"),
		},
		{
			name:        "existing long position, go more long",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			},
			side:                     v2types.Direction_LONG,
			margin:                   sdk.NewInt(1000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(2000),
			expectedOpenNotional:     sdk.NewDec(20_000),
			expectedSize:             sdk.MustNewDecFromStr("19999.999900000001"),
			expectedPositionNotional: sdk.MustNewDecFromStr("20000.000099999999"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("0.000099999999"),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.0000000200000001"),
		},
		{
			name:        "existing long position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10)),
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			},
			side:                     v2types.Direction_SHORT,
			margin:                   sdk.NewInt(500),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("999.99995000000025"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("4999.99995000000025"),
			expectedSize:             sdk.MustNewDecFromStr("4999.999974999999875"),
			expectedPositionNotional: sdk.MustNewDecFromStr("4999.999900000001"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-0.00004999999925"),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-0.00004999999975"),
			expectedMarginToVault:    sdk.ZeroDec(),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.999999990000000025"),
		},
		{
			name:        "existing long position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1060)),
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			},
			side:                     v2types.Direction_SHORT,
			margin:                   sdk.NewInt(3000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("2000.0000099999999"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("20000.000099999999"),
			expectedSize:             sdk.MustNewDecFromStr("-20000.000900000027000001"),
			expectedPositionNotional: sdk.MustNewDecFromStr("20000.000099999999"),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-0.000099999999"),
			expectedMarginToVault:    sdk.MustNewDecFromStr("1000.0001099999989"),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.9999999400000009"),
		},
		{
			name:                     "new long position just under fluctuation limit",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e12)),
			initialPosition:          nil,
			side:                     v2types.Direction_LONG,
			margin:                   sdk.NewInt(47_619_047_619),
			leverage:                 sdk.OneDec(),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(47_619_047_619),
			expectedOpenNotional:     sdk.NewDec(47_619_047_619),
			expectedSize:             sdk.MustNewDecFromStr("45454545454.502066115702477367"),
			expectedPositionNotional: sdk.NewDec(47_619_047_619),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(47_619_047_619),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.09750566893414059"),
		},
		{
			name:                     "new short position",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition:          nil,
			side:                     v2types.Direction_SHORT,
			margin:                   sdk.NewInt(1000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(1000),
			expectedOpenNotional:     sdk.NewDec(10_000),
			expectedSize:             sdk.MustNewDecFromStr("-10000.000100000001"),
			expectedPositionNotional: sdk.NewDec(10_000),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.9999999800000001"),
		},
		{
			name:        "existing short position, go more short",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			},
			side:                     v2types.Direction_SHORT,
			margin:                   sdk.NewInt(1000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(2000),
			expectedOpenNotional:     sdk.NewDec(20_000),
			expectedSize:             sdk.MustNewDecFromStr("-20000.000100000001"),
			expectedPositionNotional: sdk.MustNewDecFromStr("19999.999899999999"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("0.000100000001"),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.9999999800000001"),
		},
		{
			name:        "existing short position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10)),
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			},
			side:                     v2types.Direction_LONG,
			margin:                   sdk.NewInt(500),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("999.99994999999975"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("5000.00005000000025"),
			expectedSize:             sdk.MustNewDecFromStr("-5000.000024999999875"),
			expectedPositionNotional: sdk.MustNewDecFromStr("5000.000100000001"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-0.00005000000075"),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-0.00005000000025"),
			expectedMarginToVault:    sdk.ZeroDec(),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.000000010000000025"),
		},
		{
			name:        "existing short position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1060)),
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			},
			side:                     v2types.Direction_LONG,
			margin:                   sdk.NewInt(3000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("1999.9999899999999"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("19999.999899999999"),
			expectedSize:             sdk.MustNewDecFromStr("19999.999100000026999999"),
			expectedPositionNotional: sdk.MustNewDecFromStr("19999.999899999999"),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-0.000100000001"),
			expectedMarginToVault:    sdk.MustNewDecFromStr("1000.0000900000009"),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.0000000600000009"),
		},
		{
			name:                     "new short position just under fluctuation limit",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e12)),
			initialPosition:          nil,
			side:                     v2types.Direction_SHORT,
			margin:                   sdk.NewInt(47_619_047_619),
			leverage:                 sdk.OneDec(),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(47_619_047_619),
			expectedOpenNotional:     sdk.NewDec(47_619_047_619),
			expectedSize:             sdk.MustNewDecFromStr("-49999999999.947500000000002625"),
			expectedPositionNotional: sdk.NewDec(47_619_047_619),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(47_619_047_619),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.90702947845814059"),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Log("setup app")
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			traderAddr := testutil.AccAddress()
			exchangedSize := tc.expectedSize

			t.Log("initialize market")
			market := mock.TestMarket()
			app.PerpKeeperV2.Markets.Insert(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), *market)
			amm := mock.TestAMM()
			app.PerpKeeperV2.AMMs.Insert(ctx, amm.Pair, *amm)
			app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(amm.Pair, ctx.BlockTime()), v2types.ReserveSnapshot{
				Amm:         *amm,
				TimestampMs: ctx.BlockTime().UnixMilli(),
			})
			require.NoError(t, testapp.FundAccount(app.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(tc.initialPosition.Pair, traderAddr), *tc.initialPosition)
				exchangedSize = exchangedSize.Sub(tc.initialPosition.Size_)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			resp, err := app.PerpKeeperV2.OpenPosition(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.NoError(t, err)

			t.Log("assert position response")
			require.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), resp.Position.Pair)
			require.EqualValues(t, traderAddr.String(), resp.Position.TraderAddress)
			require.EqualValues(t, tc.expectedMargin, resp.Position.Margin, "margin")
			require.EqualValues(t, tc.expectedOpenNotional, resp.Position.OpenNotional, "open notional")
			require.EqualValues(t, tc.expectedSize, resp.Position.Size_, "position size")
			require.EqualValues(t, ctx.BlockHeight(), resp.Position.LastUpdatedBlockNumber)
			require.EqualValues(t, sdk.ZeroDec(), resp.Position.LatestCumulativePremiumFraction)
			require.EqualValues(t, tc.leverage.MulInt(tc.margin), resp.ExchangedNotionalValue)
			require.EqualValues(t, exchangedSize, resp.ExchangedPositionSize)
			require.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
			require.EqualValues(t, sdk.ZeroDec(), resp.FundingPayment)
			require.EqualValues(t, tc.expectedRealizedPnl, resp.RealizedPnl)
			require.EqualValues(t, tc.expectedUnrealizedPnl, resp.UnrealizedPnlAfter)
			require.EqualValues(t, tc.expectedMarginToVault, resp.MarginToVault)
			require.EqualValues(t, tc.expectedPositionNotional, resp.PositionNotional)

			t.Log("assert position in state")
			position, err := app.PerpKeeperV2.Positions.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr))
			require.NoError(t, err)
			require.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), position.Pair)
			require.EqualValues(t, traderAddr.String(), position.TraderAddress)
			require.EqualValues(t, tc.expectedMargin, position.Margin, "margin")
			require.EqualValues(t, tc.expectedOpenNotional, position.OpenNotional, "open notional")
			require.EqualValues(t, tc.expectedSize, position.Size_, "position size")
			require.EqualValues(t, ctx.BlockHeight(), position.LastUpdatedBlockNumber)
			require.EqualValues(t, sdk.ZeroDec(), position.LatestCumulativePremiumFraction)

			exchangedNotional := tc.leverage.MulInt(tc.margin)
			exchangeFee := market.ExchangeFeeRatio.Mul(exchangedNotional).RoundInt()
			ecosystemFundFee := market.EcosystemFundFeeRatio.Mul(exchangedNotional).RoundInt()

			testutil.RequireHasTypedEvent(t, ctx, &v2types.PositionChangedEvent{
				Pair:               asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				TraderAddress:      traderAddr.String(),
				Margin:             sdk.NewCoin(denoms.NUSD, tc.expectedMargin.RoundInt()),
				PositionNotional:   tc.expectedPositionNotional,
				ExchangedNotional:  exchangedNotional,
				ExchangedSize:      exchangedSize,
				PositionSize:       tc.expectedSize,
				RealizedPnl:        tc.expectedRealizedPnl,
				UnrealizedPnlAfter: tc.expectedUnrealizedPnl,
				BadDebt:            sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
				FundingPayment:     sdk.ZeroDec(),
				TransactionFee:     sdk.NewCoin(denoms.NUSD, exchangeFee.Add(ecosystemFundFee)),
				BlockHeight:        ctx.BlockHeight(),
				BlockTimeMs:        ctx.BlockTime().UnixMilli(),
			})
		})
	}
}

func TestOpenPositionError(t *testing.T) {
	testCases := []struct {
		name        string
		traderFunds sdk.Coins

		initialPosition *v2types.Position

		// position arguments
		side      v2types.Direction
		margin    sdk.Int
		leverage  sdk.Dec
		baseLimit sdk.Dec

		expectedErr error
	}{
		{
			name:            "not enough trader funds",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			initialPosition: nil,
			side:            v2types.Direction_LONG,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.ZeroDec(),
			expectedErr:     sdkerrors.ErrInsufficientFunds,
		},
		{
			name:        "position has bad debt",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			},
			side:        v2types.Direction_LONG,
			margin:      sdk.NewInt(1),
			leverage:    sdk.OneDec(),
			baseLimit:   sdk.ZeroDec(),
			expectedErr: v2types.ErrMarginRatioTooLow,
		},
		{
			name:            "new long position not over base limit",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            v2types.Direction_LONG,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     v2types.ErrAssetFailsUserLimit,
		},
		{
			name:            "new short position not under base limit",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            v2types.Direction_SHORT,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     v2types.ErrAssetFailsUserLimit,
		},
		{
			name:            "quote asset amount is zero",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            v2types.Direction_SHORT,
			margin:          sdk.NewInt(0),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     v2types.ErrQuoteAmountIsZero,
		},
		{
			name:            "leverage amount is zero",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            v2types.Direction_SHORT,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(0),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     v2types.ErrLeverageIsZero,
		},
		{
			name:            "leverage amount is too high - SELL",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            v2types.Direction_SHORT,
			margin:          sdk.NewInt(100),
			leverage:        sdk.NewDec(100),
			baseLimit:       sdk.NewDec(11_000),
			expectedErr:     v2types.ErrLeverageIsTooHigh,
		},
		{
			name:            "leverage amount is too high - BUY",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            v2types.Direction_LONG,
			margin:          sdk.NewInt(100),
			leverage:        sdk.NewDec(16),
			baseLimit:       sdk.NewDec(0),
			expectedErr:     v2types.ErrLeverageIsTooHigh,
		},
		{
			name:            "new long position over fluctuation limit",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e12)),
			initialPosition: nil,
			side:            v2types.Direction_LONG,
			margin:          sdk.NewInt(100_000e6),
			leverage:        sdk.OneDec(),
			baseLimit:       sdk.ZeroDec(),
			expectedErr:     v2types.ErrOverFluctuationLimit,
		},
		{
			name:            "new short position over fluctuation limit",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e12)),
			initialPosition: nil,
			side:            v2types.Direction_SHORT,
			margin:          sdk.NewInt(100_000e6),
			leverage:        sdk.OneDec(),
			baseLimit:       sdk.ZeroDec(),
			expectedErr:     v2types.ErrOverFluctuationLimit,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			traderAddr := testutil.AccAddress()

			t.Log("initialize market")
			market := mock.TestMarket()
			app.PerpKeeperV2.Markets.Insert(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), *market)
			amm := mock.TestAMM()
			app.PerpKeeperV2.AMMs.Insert(ctx, amm.Pair, *amm)
			app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(amm.Pair, ctx.BlockTime()), v2types.ReserveSnapshot{
				Amm:         *amm,
				TimestampMs: ctx.BlockTime().UnixMilli(),
			})
			require.NoError(t, testapp.FundAccount(app.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(tc.initialPosition.Pair, traderAddr), *tc.initialPosition)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			resp, err := app.PerpKeeperV2.OpenPosition(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.ErrorContains(t, err, tc.expectedErr.Error())
			require.Nil(t, resp)
		})
	}
}

func TestOpenPositionInvalidPair(t *testing.T) {
	t.Log("Setup Nibiru app, pair, and trader without a perpamm.")
	app, ctx := testapp.NewNibiruTestAppAndContext(true)
	pair := asset.MustNewPair("xxx:yyy")
	trader := testutil.AccAddress()

	t.Log("open a position on invalid 'pair'")
	side := v2types.Direction_LONG
	quote := sdk.NewInt(60)
	leverage := sdk.NewDec(10)
	baseLimit := sdk.NewDec(150)
	resp, err := app.PerpKeeperV2.OpenPosition(
		ctx, pair, side, trader, quote, leverage, baseLimit)
	require.ErrorContains(t, err, v2types.ErrPairNotFound.Error())
	require.Nil(t, resp)
}

func TestClosePosition(t *testing.T) {
	app.SetPrefixes(app.AccountAddressPrefix)
	tests := []struct {
		name string

		initialPosition    v2types.Position
		newPriceMultiplier sdk.Dec
		newLatestCPF       sdk.Dec

		expectedFundingPayment         sdk.Dec
		expectedBadDebt                sdk.Dec
		expectedRealizedPnl            sdk.Dec
		expectedMarginToVault          sdk.Dec
		expectedExchangedNotionalValue sdk.Dec
	}{
		{
			name: "long position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user closes position
			// user ends up with realized PnL of +100 NUSD, unrealized PnL after of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                    sdk.NewDec(100), // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.NewDec(2),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("199.999999980000000002"),
			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("99.999999980000000002"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-107.999999980000000002"),
		},
		{
			name: "close long position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			//   position and open notional value is 100 NUSD
			// BTC drops in value, now its price is 1 BTC = 0.95 NUSD
			// user has position notional value of 195 NUSD and unrealized PnL of -5 NUSD
			// user closes position
			// user ends up with realized PnL of -5 NUSD, unrealized PnL of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100),
				Margin:                          sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(100),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.MustNewDecFromStr("0.95"),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("-5.000000009499999999"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-2.999999990500000001"), // 10(old) + (-5)(realized PnL) - (2)(funding payment)
			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("94.999999990500000001"),
		},

		/*==========================SHORT POSITIONS===========================*/
		{
			name: "close short position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC drops in value, now its price is 1 BTC = 0.95 NUSD
			// user has position notional value of 95 NUSD and unrealized PnL of 5 NUSD
			// user closes position
			// user ends up with realized PnL of 5 NUSD, unrealized PnL of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100),
				Margin:                          sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(100),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.MustNewDecFromStr("0.95"),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(-2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("4.999999990499999999"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-16.999999990499999999"), // old(10) + (5)(realizedPnL) - (-2)(fundingPayment)
			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("95.000000009500000001"),
		},
		{
			name: "decrease short position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1 BTC = 1.05 NUSD
			// user has position notional value of 105 NUSD and unrealized PnL of -5 NUSD
			// user closes their position
			// user ends up with realized PnL of -5 NUSD, unrealized PnL of 0 NUSD
			//   position notional value of 0 NUSD
			initialPosition: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.MustNewDecFromStr("1.05"),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(-2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("-5.000000010500000001"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-6.999999989499999999"), // old(10) + (-5)(realizedPnL) - (-2)(fundingPayment)
			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("105.000000010500000001"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			traderAddr := sdk.MustAccAddressFromBech32(tc.initialPosition.TraderAddress)

			market := mock.TestMarket().WithLatestCumulativePremiumFraction(tc.newLatestCPF)
			amm := mock.TestAMM().WithPriceMultiplier(tc.newPriceMultiplier)
			app.PerpKeeperV2.Markets.Insert(ctx, tc.initialPosition.Pair, *market)
			app.PerpKeeperV2.AMMs.Insert(ctx, tc.initialPosition.Pair, *amm)
			app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(tc.initialPosition.Pair, ctx.BlockTime()), v2types.ReserveSnapshot{
				Amm:         *amm,
				TimestampMs: ctx.BlockTime().UnixMilli(),
			})
			app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(tc.initialPosition.Pair, traderAddr), tc.initialPosition)
			testapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e18)))
			testapp.FundModuleAccount(app.BankKeeper, ctx, types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e18)))

			t.Log("close position")
			resp, err := app.PerpKeeperV2.ClosePosition(
				ctx,
				tc.initialPosition.Pair,
				traderAddr,
			)

			require.NoError(t, err)
			assert.Equal(t, v2types.PositionResp{
				Position: &v2types.Position{
					TraderAddress:                   tc.initialPosition.TraderAddress,
					Pair:                            tc.initialPosition.Pair,
					Size_:                           sdk.ZeroDec(),
					Margin:                          sdk.ZeroDec(),
					OpenNotional:                    sdk.ZeroDec(),
					LatestCumulativePremiumFraction: tc.newLatestCPF,
					LastUpdatedBlockNumber:          ctx.BlockHeight(),
				},
				ExchangedNotionalValue: tc.expectedExchangedNotionalValue,
				ExchangedPositionSize:  tc.initialPosition.Size_.Neg(),
				BadDebt:                tc.expectedBadDebt,
				FundingPayment:         tc.expectedFundingPayment,
				RealizedPnl:            tc.expectedRealizedPnl,
				UnrealizedPnlAfter:     sdk.ZeroDec(),
				MarginToVault:          tc.expectedMarginToVault,
				PositionNotional:       sdk.ZeroDec(),
			}, *resp)

			testutil.RequireHasTypedEvent(t, ctx, &v2types.PositionChangedEvent{
				Pair:               tc.initialPosition.Pair,
				TraderAddress:      tc.initialPosition.TraderAddress,
				Margin:             sdk.NewInt64Coin(denoms.NUSD, 0),
				PositionNotional:   sdk.ZeroDec(),
				ExchangedNotional:  tc.expectedExchangedNotionalValue,
				ExchangedSize:      tc.initialPosition.Size_.Neg(),
				PositionSize:       sdk.ZeroDec(),
				RealizedPnl:        tc.expectedRealizedPnl,
				UnrealizedPnlAfter: sdk.ZeroDec(),
				BadDebt:            sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
				FundingPayment:     tc.expectedFundingPayment,
				TransactionFee:     sdk.NewInt64Coin(denoms.NUSD, 0),
				BlockHeight:        ctx.BlockHeight(),
				BlockTimeMs:        ctx.BlockTime().UnixMilli(),
			})
		})
	}
}

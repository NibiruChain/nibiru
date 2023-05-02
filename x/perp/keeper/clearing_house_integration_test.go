package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/oracle/integration_test/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/keeper"

	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
)

func createInitMarket() Action {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)

	return CreateCustomMarket(pairBtcUsdc,
		/* quoteReserve */ sdk.NewDec(1*common.TO_MICRO*common.TO_MICRO),
		/* baseReserve */ sdk.NewDec(1*common.TO_MICRO*common.TO_MICRO),
		perpammtypes.MarketConfig{
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.OneDec(), // 100%,
			TradeLimitRatio:        sdk.OneDec(),
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
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1020000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, perpammtypes.Direction_LONG, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						perptypes.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1_000_000),
							OpenNotional:                    sdk.NewDec(1_000_000),
							Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
							BlockNumber:                     1,
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
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(perptypes.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.NewDec(1000),
					OpenNotional:                    sdk.NewDec(10_000),
					Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
					BlockNumber:                     1,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})),
				PositionChangedEventShouldBeEqual(&perptypes.PositionChangedEvent{
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
					MarkPrice:          sdk.MustNewDecFromStr("1.0000000200000001"),
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
			SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2040000)))),
			OpenPosition(alice, pairBtcUsdc, perpammtypes.Direction_LONG, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
		).When(
			MoveToNextBlock(),
			OpenPosition(alice, pairBtcUsdc, perpammtypes.Direction_LONG, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec(),
				OpenPositionResp_PositionShouldBeEqual(
					perptypes.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(2_000_000),
						OpenNotional:                    sdk.NewDec(2_000_000),
						Size_:                           sdk.MustNewDecFromStr("19999.999600000008000000"),
						BlockNumber:                     2,
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
			PositionChangedEventShouldBeEqual(&perptypes.PositionChangedEvent{
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
				MarkPrice:          sdk.MustNewDecFromStr("1.000000040000000400"),
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

		initialPosition *perptypes.Position

		side      perpammtypes.Direction
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
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1_200_000)),
			initialPosition:          nil,
			side:                     perpammtypes.Direction_LONG,
			margin:                   sdk.NewInt(1_000_000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(1_000_000),
			expectedOpenNotional:     sdk.NewDec(10_000_000),
			expectedSize:             sdk.MustNewDecFromStr("9999900.000999990000099999"),
			expectedPositionNotional: sdk.NewDec(10_000_000),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1_000_000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.000020000100000000"),
		},
		{
			name:        "existing long position, go more long",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1_200_000)),
			initialPosition: &perptypes.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     perpammtypes.Direction_LONG,
			margin:                   sdk.NewInt(1_001_000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(1_002_000),
			expectedOpenNotional:     sdk.NewDec(10_020_000),
			expectedSize:             sdk.MustNewDecFromStr("10019899.800902992961040460"),
			expectedPositionNotional: sdk.MustNewDecFromStr("10020000.200100998998969980"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("0.200100998998969980"),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1001000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.000020020100200100"),
		},
		{
			name:        "existing long position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1_000_000)),
			initialPosition: &perptypes.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(20_000_000),
				Margin:                          sdk.NewDec(2_000_000),
				OpenNotional:                    sdk.NewDec(20_000_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     perpammtypes.Direction_SHORT,
			margin:                   sdk.NewInt(1_000_000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("1999800.001999940000999980"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("9999800.001999940000999980"),
			expectedSize:             sdk.MustNewDecFromStr("9999899.998999989999899999"),
			expectedPositionNotional: sdk.MustNewDecFromStr("9999600.007999840003199936"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-199.994000099997800044"),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-199.998000059999000020"),
			expectedMarginToVault:    sdk.ZeroDec(),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.999980000100000000"),
		},
		{
			name:        "existing long position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1_200_000)),
			initialPosition: &perptypes.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10_000_000),
				Margin:                          sdk.NewDec(1_000_000),
				OpenNotional:                    sdk.NewDec(10_000_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     perpammtypes.Direction_SHORT,
			margin:                   sdk.NewInt(3_000_000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("2000009.999900000999990000"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("20000099.999000009999900001"),
			expectedSize:             sdk.MustNewDecFromStr("-20000900.027000810024300729"),
			expectedPositionNotional: sdk.MustNewDecFromStr("20000099.999000009999900001"),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-99.999000009999900001"),
			expectedMarginToVault:    sdk.MustNewDecFromStr("1000109.998900010999890001"),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.999940000900000000"),
		},
		{
			name:                     "new long position just under fluctuation limit",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1*common.TO_MICRO*common.TO_MICRO)),
			initialPosition:          nil,
			side:                     perpammtypes.Direction_LONG,
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
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1_200_000)),
			initialPosition:          nil,
			side:                     perpammtypes.Direction_SHORT,
			margin:                   sdk.NewInt(1_000_000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(1_000_000),
			expectedOpenNotional:     sdk.NewDec(10_000_000),
			expectedSize:             sdk.MustNewDecFromStr("-10000100.001000010000100001"),
			expectedPositionNotional: sdk.NewDec(10_000_000),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1_000_000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.999980000100000000"),
		},
		{
			name:        "existing short position, go more short",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1_200_000)),
			initialPosition: &perptypes.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     perpammtypes.Direction_SHORT,
			margin:                   sdk.NewInt(1_000_000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(1_001_000),
			expectedOpenNotional:     sdk.NewDec(10_010_000),
			expectedSize:             sdk.MustNewDecFromStr("-10010100.001000010000100001"),
			expectedPositionNotional: sdk.MustNewDecFromStr("10009999.800100997001029960"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("0.199899002998970040"),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1_000_000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.999980000100000000"),
		},
		{
			name:        "existing short position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1_200_000)),
			initialPosition: &perptypes.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-20_000_000),
				Margin:                          sdk.NewDec(2_000_000),
				OpenNotional:                    sdk.NewDec(20_000_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     perpammtypes.Direction_LONG,
			margin:                   sdk.NewInt(1_000_000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("1999799.997999939998999980"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("10000200.002000060001000020"),
			expectedSize:             sdk.MustNewDecFromStr("-10000099.999000009999900001"),
			expectedPositionNotional: sdk.MustNewDecFromStr("10000400.008000160003200064"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-200.006000100002200044"),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-200.002000060001000020"),
			expectedMarginToVault:    sdk.ZeroDec(),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.000020000100000000"),
		},
		{
			name:        "existing short position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1_200_000)),
			initialPosition: &perptypes.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-10_000_000),
				Margin:                          sdk.NewDec(1_000_000),
				OpenNotional:                    sdk.NewDec(10_000_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     perpammtypes.Direction_LONG,
			margin:                   sdk.NewInt(3_000_000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("1999989.999899998999990000"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("19999899.998999989999899999"),
			expectedSize:             sdk.MustNewDecFromStr("19999100.026999190024299271"),
			expectedPositionNotional: sdk.MustNewDecFromStr("19999899.998999989999899999"),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-100.001000010000100001"),
			expectedMarginToVault:    sdk.MustNewDecFromStr("1000090.000900009000090001"),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.000060000900000000"),
		},
		{
			name:                     "new short position just under fluctuation limit",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1*common.TO_MICRO*common.TO_MICRO)),
			initialPosition:          nil,
			side:                     perpammtypes.Direction_SHORT,
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
			t.Log("Setup Nibiru app and constants")
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
			traderAddr := testutil.AccAddress()
			exchangedSize := tc.expectedSize

			t.Log("initialize market")
			assert.NoError(t, nibiruApp.PerpAmmKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* quoteReserve */ sdk.NewDec(1*common.TO_MICRO*common.TO_MICRO),
				/* baseReserve */ sdk.NewDec(1*common.TO_MICRO*common.TO_MICRO),
				perpammtypes.MarketConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(), // 100%,
					TradeLimitRatio:        sdk.OneDec(),
				},
				sdk.OneDec(),
			))
			keeper.SetPairMetadata(nibiruApp.PerpKeeper, ctx, perptypes.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("initialize trader funds")
			require.NoError(t, testapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				keeper.SetPosition(nibiruApp.PerpKeeper, ctx, *tc.initialPosition)
				exchangedSize = exchangedSize.Sub(tc.initialPosition.Size_)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			resp, err := nibiruApp.PerpKeeper.OpenPosition(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.NoError(t, err)

			t.Log("assert position response")
			require.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), resp.Position.Pair)
			require.EqualValues(t, traderAddr.String(), resp.Position.TraderAddress)
			require.EqualValues(t, tc.expectedMargin, resp.Position.Margin, "margin")
			require.EqualValues(t, tc.expectedOpenNotional, resp.Position.OpenNotional, "open notional")
			require.EqualValues(t, tc.expectedSize, resp.Position.Size_, "position size")
			require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
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
			position, err := nibiruApp.PerpKeeper.Positions.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr))
			require.NoError(t, err)
			require.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), position.Pair)
			require.EqualValues(t, traderAddr.String(), position.TraderAddress)
			require.EqualValues(t, tc.expectedMargin, position.Margin, "margin")
			require.EqualValues(t, tc.expectedOpenNotional, position.OpenNotional, "open notional")
			require.EqualValues(t, tc.expectedSize, position.Size_, "position size")
			require.EqualValues(t, ctx.BlockHeight(), position.BlockNumber)
			require.EqualValues(t, sdk.ZeroDec(), position.LatestCumulativePremiumFraction)

			exchangedNotional := tc.leverage.MulInt(tc.margin)
			feePoolFee := nibiruApp.PerpKeeper.GetParams(ctx).FeePoolFeeRatio.Mul(exchangedNotional).RoundInt()
			ecosystemFundFee := nibiruApp.PerpKeeper.GetParams(ctx).EcosystemFundFeeRatio.Mul(exchangedNotional).RoundInt()

			testutil.RequireHasTypedEvent(t, ctx, &perptypes.PositionChangedEvent{
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
				MarkPrice:          tc.expectedMarkPrice,
				FundingPayment:     sdk.ZeroDec(),
				TransactionFee:     sdk.NewCoin(denoms.NUSD, feePoolFee.Add(ecosystemFundFee)),
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

		// market params
		poolTradeLimitRatio sdk.Dec

		initialPosition *perptypes.Position

		// position arguments
		side      perpammtypes.Direction
		margin    sdk.Int
		leverage  sdk.Dec
		baseLimit sdk.Dec

		expectedErr error
	}{
		{
			name:                "not enough trader funds",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                perpammtypes.Direction_LONG,
			margin:              sdk.NewInt(1000),
			leverage:            sdk.NewDec(10),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         sdkerrors.ErrInsufficientFunds,
		},
		{
			name:                "position has bad debt",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition: &perptypes.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:        perpammtypes.Direction_LONG,
			margin:      sdk.NewInt(1),
			leverage:    sdk.OneDec(),
			baseLimit:   sdk.ZeroDec(),
			expectedErr: perptypes.ErrMarginRatioTooLow,
		},
		{
			name:                "new long position not over base limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                perpammtypes.Direction_LONG,
			margin:              sdk.NewInt(1000),
			leverage:            sdk.NewDec(10),
			baseLimit:           sdk.NewDec(10_000),
			expectedErr:         perpammtypes.ErrAssetFailsUserLimit,
		},
		{
			name:                "new short position not under base limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                perpammtypes.Direction_SHORT,
			margin:              sdk.NewInt(1000),
			leverage:            sdk.NewDec(10),
			baseLimit:           sdk.NewDec(10_000),
			expectedErr:         perpammtypes.ErrAssetFailsUserLimit,
		},
		{
			name:                "quote asset amount is zero",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                perpammtypes.Direction_SHORT,
			margin:              sdk.NewInt(0),
			leverage:            sdk.NewDec(10),
			baseLimit:           sdk.NewDec(10_000),
			expectedErr:         perptypes.ErrQuoteAmountIsZero,
		},
		{
			name:                "leverage amount is zero",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                perpammtypes.Direction_SHORT,
			margin:              sdk.NewInt(1000),
			leverage:            sdk.NewDec(0),
			baseLimit:           sdk.NewDec(10_000),
			expectedErr:         perptypes.ErrLeverageIsZero,
		},
		{
			name:                "leverage amount is too high - SELL",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                perpammtypes.Direction_SHORT,
			margin:              sdk.NewInt(100),
			leverage:            sdk.NewDec(100),
			baseLimit:           sdk.NewDec(11_000),
			expectedErr:         perptypes.ErrLeverageIsTooHigh,
		},
		{
			name:                "leverage amount is too high - BUY",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                perpammtypes.Direction_LONG,
			margin:              sdk.NewInt(100),
			leverage:            sdk.NewDec(16),
			baseLimit:           sdk.NewDec(0),
			expectedErr:         perptypes.ErrLeverageIsTooHigh,
		},
		{
			name:                "new long position over fluctuation limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1*common.TO_MICRO*common.TO_MICRO)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                perpammtypes.Direction_LONG,
			margin:              sdk.NewInt(100_000 * common.TO_MICRO),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         perpammtypes.ErrOverFluctuationLimit,
		},
		{
			name:                "new short position over fluctuation limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1*common.TO_MICRO*common.TO_MICRO)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                perpammtypes.Direction_SHORT,
			margin:              sdk.NewInt(100_000 * common.TO_MICRO),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         perpammtypes.ErrOverFluctuationLimit,
		},
		{
			name:                "new long position over trade limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10_000*common.TO_MICRO)),
			poolTradeLimitRatio: sdk.MustNewDecFromStr("0.01"),
			initialPosition:     nil,
			side:                perpammtypes.Direction_LONG,
			margin:              sdk.NewInt(100_000 * common.TO_MICRO),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         perpammtypes.ErrOverTradingLimit,
		},
		{
			name:                "new short position over trade limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10_000*common.TO_MICRO)),
			poolTradeLimitRatio: sdk.MustNewDecFromStr("0.01"),
			initialPosition:     nil,
			side:                perpammtypes.Direction_SHORT,
			margin:              sdk.NewInt(100_000 * common.TO_MICRO),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         perpammtypes.ErrOverTradingLimit,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Setup Nibiru app and constants")
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
			traderAddr := testutil.AccAddress()

			t.Log("initialize market")
			assert.NoError(t, nibiruApp.PerpAmmKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* tradeLimitRatio */
				/* quoteReserve */
				sdk.NewDec(1*common.TO_MICRO*common.TO_MICRO),
				/* baseReserve */ sdk.NewDec(1*common.TO_MICRO*common.TO_MICRO),
				perpammtypes.MarketConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(), // 100%,
					TradeLimitRatio:        tc.poolTradeLimitRatio,
				},
				sdk.OneDec(),
			))
			keeper.SetPairMetadata(nibiruApp.PerpKeeper, ctx, perptypes.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("initialize trader funds")
			require.NoError(t, testapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				keeper.SetPosition(nibiruApp.PerpKeeper, ctx, *tc.initialPosition)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			resp, err := nibiruApp.PerpKeeper.OpenPosition(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.ErrorContains(t, err, tc.expectedErr.Error())
			require.Nil(t, resp)
		})
	}
}

func TestOpenPositionInvalidPair(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "open pos - uninitialized pool raised pair not supported error",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader without a perpamm.")
				nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
				pair := asset.MustNewPair("xxx:yyy")

				trader := testutil.AccAddress()

				t.Log("open a position on invalid 'pair'")
				side := perpammtypes.Direction_LONG
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				resp, err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)
				require.ErrorContains(t, err, perptypes.ErrPairNotFound.Error())
				require.Nil(t, resp)
			},
		},
		{
			name: "open pos - market not set on the perp PairMetadata ",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
				pair := asset.MustNewPair("xxx:yyy")

				t.Log("Set market defined by pair on PerpAmmKeeper")
				perpammKeeper := &nibiruApp.PerpAmmKeeper
				require.NoError(t, nibiruApp.PerpAmmKeeper.CreatePool(
					ctx,
					pair,
					sdk.NewDec(5*common.TO_MICRO), //
					sdk.NewDec(5*common.TO_MICRO), // 5 tokens
					perpammtypes.MarketConfig{
						FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
						MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
						MaxLeverage:            sdk.MustNewDecFromStr("15"),
						MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"), // 100%,
						TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					},
					sdk.NewDec(2),
				))

				require.True(t, perpammKeeper.ExistsPool(ctx, pair))

				t.Log("Attempt to open long position (expected unsuccessful)")
				trader := testutil.AccAddress()
				side := perpammtypes.Direction_LONG
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				resp, err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)
				require.ErrorIs(t, err, collections.ErrNotFound)
				require.Nil(t, resp)
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

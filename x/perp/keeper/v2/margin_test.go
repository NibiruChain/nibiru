package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	. "github.com/NibiruChain/nibiru/x/oracle/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion/v2"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestAddMargin(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("existing long position, add margin").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2020)))),
				OpenPosition(alice, pairBtcUsdc, v2types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				AddMargin(alice, pairBtcUsdc, sdk.NewInt(1000)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(v2types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("9999.999900000001000000"),
					Margin:                          sdk.NewDec(2000),
					OpenNotional:                    sdk.NewDec(10000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&v2types.PositionChangedEvent{
					Pair:               pairBtcUsdc,
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
				BalanceEqual(alice, denoms.USDC, sdk.ZeroInt()),
				ModuleBalanceEqual(v2types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(10)),
				ModuleBalanceEqual(v2types.FeePoolModuleAccount, denoms.USDC, sdk.NewInt(10)),
			),

		TC("existing short position, add margin").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2020)))),
				OpenPosition(alice, pairBtcUsdc, v2types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				AddMargin(alice, pairBtcUsdc, sdk.NewInt(1000)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(v2types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
					Margin:                          sdk.NewDec(2000),
					OpenNotional:                    sdk.NewDec(10000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&v2types.PositionChangedEvent{
					Pair:               pairBtcUsdc,
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
				BalanceEqual(alice, denoms.USDC, sdk.ZeroInt()),
				ModuleBalanceEqual(v2types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(10)),
				ModuleBalanceEqual(v2types.FeePoolModuleAccount, denoms.USDC, sdk.NewInt(10)),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestRemoveMargin(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("existing long position, remove margin").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1002)))),
				OpenPosition(alice, pairBtcUsdc, v2types.Direction_LONG, sdk.NewInt(1000), sdk.OneDec(), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				RemoveMargin(alice, pairBtcUsdc, sdk.NewInt(500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(v2types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("999.999999000000001000"),
					Margin:                          sdk.NewDec(500),
					OpenNotional:                    sdk.NewDec(1000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&v2types.PositionChangedEvent{
					Pair:               pairBtcUsdc,
					TraderAddress:      alice.String(),
					Margin:             sdk.NewCoin(denoms.USDC, sdk.NewInt(500)),
					PositionNotional:   sdk.NewDec(1000),
					ExchangedNotional:  sdk.ZeroDec(),
					ExchangedSize:      sdk.ZeroDec(),
					PositionSize:       sdk.MustNewDecFromStr("999.999999000000001000"),
					RealizedPnl:        sdk.ZeroDec(),
					UnrealizedPnlAfter: sdk.ZeroDec(),
					BadDebt:            sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					FundingPayment:     sdk.ZeroDec(),
					TransactionFee:     sdk.NewCoin(denoms.USDC, sdk.NewInt(2)), // 20 bps
					BlockHeight:        2,
					BlockTimeMs:        startBlockTime.Add(time.Second*5).UnixNano() / 1e6,
				}),
				BalanceEqual(alice, denoms.USDC, sdk.NewInt(500)),
				ModuleBalanceEqual(v2types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(1)),
				ModuleBalanceEqual(v2types.FeePoolModuleAccount, denoms.USDC, sdk.NewInt(1)),
			),

		TC("existing short position, remove margin").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1002)))),
				OpenPosition(alice, pairBtcUsdc, v2types.Direction_SHORT, sdk.NewInt(1000), sdk.OneDec(), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				RemoveMargin(alice, pairBtcUsdc, sdk.NewInt(500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(v2types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("-1000.000001000000001000"),
					Margin:                          sdk.NewDec(500),
					OpenNotional:                    sdk.NewDec(1000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&v2types.PositionChangedEvent{
					Pair:               pairBtcUsdc,
					TraderAddress:      alice.String(),
					Margin:             sdk.NewCoin(denoms.USDC, sdk.NewInt(500)),
					PositionNotional:   sdk.NewDec(1000),
					ExchangedNotional:  sdk.ZeroDec(),
					ExchangedSize:      sdk.ZeroDec(),
					PositionSize:       sdk.MustNewDecFromStr("-1000.000001000000001000"),
					RealizedPnl:        sdk.ZeroDec(),
					UnrealizedPnlAfter: sdk.ZeroDec(),
					BadDebt:            sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					FundingPayment:     sdk.ZeroDec(),
					TransactionFee:     sdk.NewCoin(denoms.USDC, sdk.NewInt(2)), // 20 bps
					BlockHeight:        2,
					BlockTimeMs:        startBlockTime.Add(time.Second*5).UnixNano() / 1e6,
				}),
				BalanceEqual(alice, denoms.USDC, sdk.NewInt(500)),
				ModuleBalanceEqual(v2types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(1)),
				ModuleBalanceEqual(v2types.FeePoolModuleAccount, denoms.USDC, sdk.NewInt(1)),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

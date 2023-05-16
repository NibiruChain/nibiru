package keeper_test

import (
	"testing"
	"time"

	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion/v1"
	. "github.com/NibiruChain/nibiru/x/perp/v1/amm/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v1/amm/integration/assertion"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/oracle/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v1"
	"github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
)

func createInitMarket() Action {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)

	return CreateCustomMarket(pairBtcUsdc,
		/* quoteReserve */ sdk.NewDec(1*common.TO_MICRO*common.TO_MICRO),
		/* baseReserve */ sdk.NewDec(1*common.TO_MICRO*common.TO_MICRO),
		types.MarketConfig{
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.OneDec(), // 100%,
			TradeLimitRatio:        sdk.OneDec(),
		})
}

func TestBiasChangeOnMarket(t *testing.T) {
	alice, bob := testutil.AccAddress(), testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("simple open long position").
			Given(
				createInitMarket(),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1200000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc,
					Market_BiasShouldBeEqualTo(sdk.MustNewDecFromStr("9999900.000999990000099999")), // Bias equal to PositionSize
				),
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionSizeShouldBeEqualTo(sdk.MustNewDecFromStr("9999900.000999990000099999"))),
			),

		TC("additional long position").
			Given(
				createInitMarket(),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2200000)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc,
					Market_BiasShouldBeEqualTo(sdk.MustNewDecFromStr("19999600.007999840003199936")), // Bias equal to PositionSize
				),
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionSizeShouldBeEqualTo(sdk.MustNewDecFromStr("19999600.007999840003199936"))),
			),
		TC("simple open short position").
			Given(
				createInitMarket(),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1200000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc,
					Market_BiasShouldBeEqualTo(sdk.MustNewDecFromStr("-10000100.001000010000100001")), // Bias equal to PositionSize
				),
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionSizeShouldBeEqualTo(sdk.MustNewDecFromStr("-10000100.001000010000100001"))),
			),

		TC("additional short position").
			Given(
				createInitMarket(),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2200000)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc,
					Market_BiasShouldBeEqualTo(sdk.MustNewDecFromStr("-20000400.008000160003200064")), // Bias equal to PositionSize
				),
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionSizeShouldBeEqualTo(sdk.MustNewDecFromStr("-20000400.008000160003200064"))),
			),
		TC("open long position and close it").
			Given(
				createInitMarket(),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2200000)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc,
					Market_BiasShouldBeEqualTo(sdk.ZeroDec()), // Bias equal to PositionSize
				),
				PositionShouldNotExist(alice, pairBtcUsdc),
			),
		TC("open long position and open smaller short").
			Given(
				createInitMarket(),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(12200000)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(10000000), sdk.NewDec(10), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc,
					Market_BiasShouldBeEqualTo(sdk.MustNewDecFromStr("89991900.728934395904368607")), // Bias equal to PositionSize
				),
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionSizeShouldBeEqualTo(sdk.MustNewDecFromStr("89991900.728934395904368607"))),
			),

		TC("2 positions, one long, one short with same amount should set Bias to 0").
			Given(
				createInitMarket(),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1200000)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1200000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
				OpenPosition(bob, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc,
					Market_BiasShouldBeEqualTo(sdk.ZeroDec()), // Bias equal to PositionSize
				),
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionSizeShouldBeEqualTo(sdk.MustNewDecFromStr("9999900.000999990000099999"))),
				PositionShouldBeEqual(bob, pairBtcUsdc, Position_PositionSizeShouldBeEqualTo(sdk.MustNewDecFromStr("-9999900.000999990000099999"))),
			),

		TC("Open long position and liquidate").
			Given(
				createInitMarket(),
				SetLiquidator(bob),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.MustNewDecFromStr("2.1")),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1200000)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1200000)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000000), sdk.NewDec(10), sdk.ZeroDec()),
				MoveToNextBlock(),
				ChangeMaintenanceMarginRatio(pairBtcUsdc, sdk.MustNewDecFromStr("0.2")),
				ChangeLiquidationFeeRatio(sdk.MustNewDecFromStr("0.2")),
			).
			When(
				LiquidatePosition(bob, alice, pairBtcUsdc),
			).Then(
			MarketShouldBeEqual(pairBtcUsdc,
				Market_BiasShouldBeEqualTo(sdk.ZeroDec()), // Bias equal to PositionSize
			),
			PositionShouldNotExist(alice, pairBtcUsdc),
		),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

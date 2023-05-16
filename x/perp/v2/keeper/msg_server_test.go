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
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion/v2"
	v2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestMsgServerOpenPosition(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("open long position").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
			).
			When(
				MsgServerOpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroInt()),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(v2types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.999999999999"),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.NewDec(1),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
					}),
				),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(99)),
			),

		TC("open short position").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
			).
			When(
				MsgServerOpenPosition(alice, pair, v2types.Direction_SHORT, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroInt()),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(v2types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("-1.000000000001"),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.NewDec(1),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
					}),
				),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(99)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerClosePosition(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("close long position").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				OpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerClosePosition(alice, pair),
			).
			Then(
				PositionShouldNotExist(alice, pair),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(100)),
			),

		TC("close short position").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				OpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerClosePosition(alice, pair),
			).
			Then(
				PositionShouldNotExist(alice, pair),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(100)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerAddMargin(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("add margin").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				OpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerAddMargin(alice, pair, sdk.NewInt(1)),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(v2types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.999999999999"),
						Margin:                          sdk.NewDec(2),
						OpenNotional:                    sdk.NewDec(1),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					}),
				),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(98)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerRemoveMargin(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("add margin").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				OpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(2), sdk.NewDec(1), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerRemoveMargin(alice, pair, sdk.NewInt(1)),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(v2types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("1.999999999996"),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.NewDec(2),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					}),
				),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(99)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerDonateToPerpEf(t *testing.T) {
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("success").
			Given(
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
			).
			When(
				MsgServerDonateToPerpEf(alice, sdk.NewInt(50)),
			).
			Then(
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(50)),
				ModuleBalanceEqual(v2types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(50)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerMultiLiquidate(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	alice := testutil.AccAddress()
	liquidator := testutil.AccAddress()
	startTime := time.Now()

	tests := TestCases{
		TC("partial liquidation").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10400))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				MoveToNextBlock(),
				MsgServerMultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(v2types.VaultModuleAccount, denoms.USDC, sdk.NewInt(750)),
				ModuleBalanceEqual(v2types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(125)),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(125)),
				PositionShouldBeEqual(alice, pairBtcUsdc,
					Position_PositionShouldBeEqualTo(
						v2types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(5000),
							Margin:                          sdk.MustNewDecFromStr("549.999951250000493750"),
							OpenNotional:                    sdk.MustNewDecFromStr("5199.999975000000375000"),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          2,
						},
					),
				),
			),

		TC("full liquidation").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10600))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				MoveToNextBlock(),
				MsgServerMultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(v2types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				ModuleBalanceEqual(v2types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				PositionShouldNotExist(alice, pairBtcUsdc),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

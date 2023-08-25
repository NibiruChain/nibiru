package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	action "github.com/NibiruChain/nibiru/x/common/testutil/action"
	assertion "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	perpaction "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	perpassertion "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"

	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestMultiLiquidate(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	pairEthUsdc := asset.Registry.Pair(denoms.ETH, denoms.USDC)
	pairAtomUsdc := asset.Registry.Pair(denoms.ATOM, denoms.USDC)
	pairSolUsdc := asset.Registry.Pair(denoms.SOL, denoms.USDC)

	alice := testutil.AccAddress()
	bob := testutil.AccAddress()
	liquidator := testutil.AccAddress()
	startTime := time.Now()

	tc := action.TestCases{
		action.TC("partial liquidation").
			Given(
				action.SetBlockNumber(1),
				action.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.InsertPosition(
					perpaction.WithTrader(alice),
					perpaction.WithPair(pairBtcUsdc),
					perpaction.WithSize(sdk.NewDec(10000)),
					perpaction.WithMargin(sdk.NewDec(1000)),
					perpaction.WithOpenNotional(sdk.NewDec(10400))),
				action.FundModule(
					types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000)),
				),
			).
			When(
				action.MoveToNextBlock(),
				perpaction.MultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(750)),
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(125)),
				assertion.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(125)),
				perpassertion.PositionShouldBeEqual(alice, pairBtcUsdc,
					perpassertion.Position_PositionShouldBeEqualTo(
						types.Position{
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

		action.TC("full liquidation").
			Given(
				action.SetBlockNumber(1),
				action.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.InsertPosition(
					perpaction.WithTrader(alice),
					perpaction.WithPair(pairBtcUsdc),
					perpaction.WithSize(sdk.NewDec(10000)),
					perpaction.WithMargin(sdk.NewDec(1000)),
					perpaction.WithOpenNotional(sdk.NewDec(10600))),
				action.FundModule(
					types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000)),
				),
			).
			When(
				action.MoveToNextBlock(),

				perpaction.MultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
					perpaction.PairTraderTuple{Pair: pairAtomUsdc, Trader: alice, Successful: false},
				),
			).
			Then(
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				assertion.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				perpassertion.PositionShouldNotExist(alice, pairBtcUsdc),
			),

		action.TC("full liquidation").
			Given(
				action.SetBlockNumber(1),
				action.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairBtcUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10600))),
				action.FundModule(
					types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000)),
				),
			).
			When(
				action.MoveToNextBlock(),
				perpaction.MultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				assertion.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				perpassertion.PositionShouldNotExist(alice, pairBtcUsdc),
			),

		action.TC("one fail liquidation - one correct").
			Given(
				action.SetBlockNumber(1),
				action.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairBtcUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10600))),
				action.FundModule(
					types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000)),
				),
			).
			When(
				action.MoveToNextBlock(),
				perpaction.MultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Successful: false},
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				assertion.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				perpassertion.PositionShouldNotExist(alice, pairBtcUsdc),
			),

		action.TC("one fail liquidation because market does not exists- one correct").
			Given(
				action.SetBlockNumber(1),
				action.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairBtcUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10600))),
				action.FundModule(
					types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000)),
				),
			).
			When(
				action.MoveToNextBlock(),
				perpaction.MultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: asset.MustNewPair("luna:usdt"), Successful: false},
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				assertion.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				perpassertion.PositionShouldNotExist(alice, pairBtcUsdc),
			),

		action.TC("realizes bad debt").
			Given(
				action.SetBlockNumber(1),
				action.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairBtcUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10800))),
				action.FundModule(
					types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000)),
				),
				action.FundModule(
					types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 50)),
				),
			).
			When(
				action.MoveToNextBlock(),
				perpaction.MultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(800)),
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.ZeroInt()),
				assertion.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				perpassertion.PositionShouldNotExist(alice, pairBtcUsdc),
			),

		action.TC("uses prepaid bad debt").
			Given(
				action.SetBlockNumber(1),
				action.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc, perpaction.WithPrepaidBadDebt(sdk.NewInt(50))),
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairBtcUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10800))),
				action.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				action.MoveToNextBlock(),
				perpaction.MultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(750)),
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.ZeroInt()),
				assertion.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				perpassertion.PositionShouldNotExist(alice, pairBtcUsdc),
				perpassertion.MarketShouldBeEqual(
					pairBtcUsdc,
					perpassertion.Market_PrepaidBadDebtShouldBeEqualTo(sdk.ZeroInt()),
				),
			),

		action.TC("healthy position").
			Given(
				action.SetBlockNumber(1),
				action.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairBtcUsdc), perpaction.WithSize(sdk.NewDec(100)), perpaction.WithMargin(sdk.NewDec(10)), perpaction.WithOpenNotional(sdk.NewDec(100))),
				action.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 10))),
			).
			When(
				action.MoveToNextBlock(),
				perpaction.MultiLiquidate(liquidator, true,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: false},
				),
			).
			Then(
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(10)),
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.ZeroInt()),
				assertion.BalanceEqual(liquidator, denoms.USDC, sdk.ZeroInt()),
				perpassertion.PositionShouldBeEqual(alice, pairBtcUsdc,
					perpassertion.Position_PositionShouldBeEqualTo(
						types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(100),
							Margin:                          sdk.NewDec(10),
							OpenNotional:                    sdk.NewDec(100),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						},
					),
				),
				perpassertion.ContainsLiquidateEvent(&types.LiquidationFailedEvent{
					Pair:       pairBtcUsdc,
					Trader:     alice.String(),
					Liquidator: liquidator.String(),
					Reason:     types.LiquidationFailedEvent_POSITION_HEALTHY,
				}),
			),

		action.TC("mixed bag").
			Given(
				action.SetBlockNumber(1),
				action.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.CreateCustomMarket(pairEthUsdc),
				perpaction.CreateCustomMarket(pairAtomUsdc),
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairBtcUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10400))),  // partial
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairEthUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10600))),  // full
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairAtomUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10000))), // healthy
				action.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 3000))),
			).
			When(
				action.MoveToNextBlock(),
				perpaction.MultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
					perpaction.PairTraderTuple{Pair: pairEthUsdc, Trader: alice, Successful: true},
					perpaction.PairTraderTuple{Pair: pairAtomUsdc, Trader: alice, Successful: false},
					perpaction.PairTraderTuple{Pair: pairSolUsdc, Trader: alice, Successful: false}, // non-existent market
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: bob, Successful: false},   // non-existent position
				),
			).
			Then(
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(2350)),
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(275)),
				assertion.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(375)),
				perpassertion.PositionShouldBeEqual(alice, pairBtcUsdc,
					perpassertion.Position_PositionShouldBeEqualTo(
						types.Position{
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
				perpassertion.PositionShouldNotExist(alice, pairEthUsdc),
				perpassertion.PositionShouldBeEqual(alice, pairAtomUsdc,
					perpassertion.Position_PositionShouldBeEqualTo(
						types.Position{
							Pair:                            pairAtomUsdc,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10000),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(10000),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						},
					),
				),

				perpassertion.ContainsLiquidateEvent(&types.LiquidationFailedEvent{
					Pair:       pairAtomUsdc,
					Trader:     alice.String(),
					Liquidator: liquidator.String(),
					Reason:     types.LiquidationFailedEvent_POSITION_HEALTHY,
				}),
				perpassertion.ContainsLiquidateEvent(&types.LiquidationFailedEvent{
					Pair:       pairSolUsdc,
					Trader:     alice.String(),
					Liquidator: liquidator.String(),
					Reason:     types.LiquidationFailedEvent_NONEXISTENT_PAIR,
				}),
				perpassertion.ContainsLiquidateEvent(&types.LiquidationFailedEvent{
					Pair:       pairBtcUsdc,
					Trader:     bob.String(),
					Liquidator: liquidator.String(),
					Reason:     types.LiquidationFailedEvent_NONEXISTENT_POSITION,
				}),
			),
	}

	action.NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestPrettyLiquidateResponse(t *testing.T) {
	type TestCase struct {
		name        string
		resps       []*types.MsgMultiLiquidateResponse_LiquidationResponse
		shouldError bool
		// prettyContains: sections of JSON string expected to be contained in
		// the pretty output.
		prettyContains []string
	}

	dummy := struct {
		LiquidatorFee sdk.Coin
		PerpEfFee     sdk.Coin
		Trader        string
	}{
		LiquidatorFee: sdk.NewInt64Coin("unibi", 420),
		PerpEfFee:     sdk.NewInt64Coin("unibi", 420),
		Trader:        "dummytrader",
	}

	testCases := []TestCase{
		{
			name:        "empty",
			resps:       []*types.MsgMultiLiquidateResponse_LiquidationResponse{},
			shouldError: false,
		},

		{
			name: "success only",
			resps: []*types.MsgMultiLiquidateResponse_LiquidationResponse{
				{
					Success:       true,
					LiquidatorFee: &dummy.LiquidatorFee,
					PerpEfFee:     &dummy.PerpEfFee,
					Trader:        dummy.Trader,
				},
			},
			shouldError: false,
			prettyContains: []string{
				`success": true`,
				`liquidator_fee": {`,
				`perp_ef_fee": {`,
				`denom": "unibi`,
				`amount": "420`,
				`trader": "dummytrader"`,
			},
		},

		{
			name: "errors only",
			resps: []*types.MsgMultiLiquidateResponse_LiquidationResponse{
				{
					Success: false,
					Error:   "failed liquidation A",
					Trader:  dummy.Trader,
				},
				{
					Success: false,
					Error:   "failed liquidation B",
					Trader:  dummy.Trader,
				},
			},
			shouldError: false,
			prettyContains: []string{
				`success": false`,
				`liquidator_fee": null`,
				`perp_ef_fee": null`,
				`trader": "dummytrader"`,
				`error": "failed liquidation A"`,
				`error": "failed liquidation B"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pretty, err := keeper.PrettyLiquidateResponse(tc.resps)

			if tc.shouldError {
				require.Errorf(t, err, "pretty: %s", pretty)
			} else {
				require.NoErrorf(t, err, "pretty: %s", pretty)
			}

			for _, prettyContains := range tc.prettyContains {
				require.Contains(t, pretty, prettyContains)
			}
		})
	}
}

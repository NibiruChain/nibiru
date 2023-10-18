package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"

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

	tc := TestCases{
		TC("partial liquidation").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10400))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				MoveToNextBlock(),
				MultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(750)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(125)),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(125)),
				PositionShouldBeEqual(alice, pairBtcUsdc,
					Position_PositionShouldBeEqualTo(
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

		TC("full liquidation").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10600))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				MoveToNextBlock(),
				MultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
					PairTraderTuple{Pair: pairAtomUsdc, Trader: alice, Successful: false},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				PositionShouldNotExist(alice, pairBtcUsdc, 1),
			),

		TC("full liquidation").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10600))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				MoveToNextBlock(),
				MultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				PositionShouldNotExist(alice, pairBtcUsdc, 1),
			),

		TC("one fail liquidation - one correct").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10600))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				MoveToNextBlock(),
				MultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Successful: false},
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				PositionShouldNotExist(alice, pairBtcUsdc, 1),
			),

		TC("one fail liquidation because market does not exists- one correct").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10600))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				MoveToNextBlock(),
				MultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: asset.MustNewPair("luna:usdt"), Successful: false},
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				PositionShouldNotExist(alice, pairBtcUsdc, 1),
			),

		TC("realizes bad debt").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10800))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 50))),
			).
			When(
				MoveToNextBlock(),
				MultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(800)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.ZeroInt()),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				PositionShouldNotExist(alice, pairBtcUsdc, 1),
			),

		TC("uses prepaid bad debt").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc, WithPrepaidBadDebt(sdk.NewInt(50))),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10800))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				MoveToNextBlock(),
				MultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(750)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.ZeroInt()),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				PositionShouldNotExist(alice, pairBtcUsdc, 1),
				MarketShouldBeEqual(pairBtcUsdc, Market_PrepaidBadDebtShouldBeEqualTo(sdk.ZeroInt())),
			),

		TC("healthy position").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(100)), WithMargin(sdk.NewDec(10)), WithOpenNotional(sdk.NewDec(100))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 10))),
			).
			When(
				MoveToNextBlock(),
				MultiLiquidate(liquidator, true,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: false},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(10)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.ZeroInt()),
				BalanceEqual(liquidator, denoms.USDC, sdk.ZeroInt()),
				PositionShouldBeEqual(alice, pairBtcUsdc,
					Position_PositionShouldBeEqualTo(
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
				ContainsLiquidateEvent(&types.LiquidationFailedEvent{
					Pair:       pairBtcUsdc,
					Trader:     alice.String(),
					Liquidator: liquidator.String(),
					Reason:     types.LiquidationFailedEvent_POSITION_HEALTHY,
				}),
			),

		TC("mixed bag").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),
				CreateCustomMarket(pairEthUsdc),
				CreateCustomMarket(pairAtomUsdc),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10400))),  // partial
				InsertPosition(WithTrader(alice), WithPair(pairEthUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10600))),  // full
				InsertPosition(WithTrader(alice), WithPair(pairAtomUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10000))), // healthy
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 3000))),
			).
			When(
				MoveToNextBlock(),
				MultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
					PairTraderTuple{Pair: pairEthUsdc, Trader: alice, Successful: true},
					PairTraderTuple{Pair: pairAtomUsdc, Trader: alice, Successful: false},
					PairTraderTuple{Pair: pairSolUsdc, Trader: alice, Successful: false}, // non-existent market
					PairTraderTuple{Pair: pairBtcUsdc, Trader: bob, Successful: false},   // non-existent position
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(2350)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(275)),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(375)),
				PositionShouldBeEqual(alice, pairBtcUsdc,
					Position_PositionShouldBeEqualTo(
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
				PositionShouldNotExist(alice, pairEthUsdc, 1),
				PositionShouldBeEqual(alice, pairAtomUsdc,
					Position_PositionShouldBeEqualTo(
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

				ContainsLiquidateEvent(&types.LiquidationFailedEvent{
					Pair:       pairAtomUsdc,
					Trader:     alice.String(),
					Liquidator: liquidator.String(),
					Reason:     types.LiquidationFailedEvent_POSITION_HEALTHY,
				}),
				ContainsLiquidateEvent(&types.LiquidationFailedEvent{
					Pair:       pairSolUsdc,
					Trader:     alice.String(),
					Liquidator: liquidator.String(),
					Reason:     types.LiquidationFailedEvent_NONEXISTENT_PAIR,
				}),
				ContainsLiquidateEvent(&types.LiquidationFailedEvent{
					Pair:       pairBtcUsdc,
					Trader:     bob.String(),
					Liquidator: liquidator.String(),
					Reason:     types.LiquidationFailedEvent_NONEXISTENT_POSITION,
				}),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
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

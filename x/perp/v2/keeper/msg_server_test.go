package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	tutilaction "github.com/NibiruChain/nibiru/x/common/testutil/action"
	tutilassert "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpaction "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	perpassert "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestMsgServerMarketOrder(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := tutilaction.TestCases{
		tutilaction.TC("open long position").
			Given(
				perpaction.CreateCustomMarket(pair),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
			).
			When(
				perpaction.MsgServerMarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroInt()),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pair,
					perpassert.Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.999999999999"),
						Margin:                          sdk.OneDec(),
						OpenNotional:                    sdk.OneDec(),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
					}),
				),
				tutilassert.BalanceEqual(alice, denoms.NUSD, sdk.NewInt(99)),
			),

		tutilaction.TC("open short position").
			Given(
				perpaction.CreateCustomMarket(pair),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
			).
			When(
				perpaction.MsgServerMarketOrder(alice, pair, types.Direction_SHORT, sdk.OneInt(), sdk.OneDec(), sdk.ZeroInt()),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pair,
					perpassert.Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("-1.000000000001"),
						Margin:                          sdk.OneDec(),
						OpenNotional:                    sdk.OneDec(),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
					}),
				),
				tutilassert.BalanceEqual(alice, denoms.NUSD, sdk.NewInt(99)),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerClosePosition(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := tutilaction.TestCases{
		tutilaction.TC("close long position").
			Given(
				perpaction.CreateCustomMarket(pair),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				perpaction.MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				tutilaction.MoveToNextBlock(),
			).
			When(
				perpaction.MsgServerClosePosition(alice, pair),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pair),
				tutilassert.BalanceEqual(alice, denoms.NUSD, sdk.NewInt(100)),
			),

		tutilaction.TC("close short position").
			Given(
				perpaction.CreateCustomMarket(pair),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				perpaction.MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				tutilaction.MoveToNextBlock(),
			).
			When(
				perpaction.MsgServerClosePosition(alice, pair),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pair),
				tutilassert.BalanceEqual(alice, denoms.NUSD, sdk.NewInt(100)),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerAddMargin(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := tutilaction.TestCases{
		tutilaction.TC("add margin").
			Given(
				perpaction.CreateCustomMarket(pair),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				perpaction.MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				tutilaction.MoveToNextBlock(),
			).
			When(
				perpaction.MsgServerAddMargin(alice, pair, sdk.OneInt()),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pair,
					perpassert.Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.999999999999"),
						Margin:                          sdk.NewDec(2),
						OpenNotional:                    sdk.OneDec(),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					}),
				),
				tutilassert.BalanceEqual(alice, denoms.NUSD, sdk.NewInt(98)),
			),
		tutilaction.TC("partial close").
			Given(
				perpaction.CreateCustomMarket(pair),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				perpaction.MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				tutilaction.MoveToNextBlock(),
			).
			When(
				perpaction.MsgServerAddMargin(alice, pair, sdk.OneInt()),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pair,
					perpassert.Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.999999999999"),
						Margin:                          sdk.NewDec(2),
						OpenNotional:                    sdk.OneDec(),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					}),
				),
				tutilassert.BalanceEqual(alice, denoms.NUSD, sdk.NewInt(98)),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerRemoveMargin(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := tutilaction.TestCases{
		tutilaction.TC("add margin").
			Given(
				perpaction.CreateCustomMarket(pair),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				perpaction.MarketOrder(alice, pair, types.Direction_LONG, sdk.NewInt(2), sdk.OneDec(), sdk.ZeroDec()),
				tutilaction.MoveToNextBlock(),
			).
			When(
				perpaction.MsgServerRemoveMargin(alice, pair, sdk.OneInt()),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pair,
					perpassert.Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("1.999999999996"),
						Margin:                          sdk.OneDec(),
						OpenNotional:                    sdk.NewDec(2),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					}),
				),
				tutilassert.BalanceEqual(alice, denoms.NUSD, sdk.NewInt(99)),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerDonateToPerpEf(t *testing.T) {
	alice := testutil.AccAddress()

	tests := tutilaction.TestCases{
		tutilaction.TC("success").
			Given(
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
			).
			When(
				perpaction.MsgServerDonateToPerpEf(alice, sdk.NewInt(50)),
			).
			Then(
				tutilassert.BalanceEqual(alice, denoms.NUSD, sdk.NewInt(50)),
				tutilassert.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(50)),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerMultiLiquidate(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	alice := testutil.AccAddress()
	liquidator := testutil.AccAddress()
	startTime := time.Now()

	tests := tutilaction.TestCases{
		tutilaction.TC("partial liquidation").
			Given(
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairBtcUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10400))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MsgServerMultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				tutilassert.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(750)),
				tutilassert.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(125)),
				tutilassert.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(125)),
				perpassert.PositionShouldBeEqual(alice, pairBtcUsdc,
					perpassert.Position_PositionShouldBeEqualTo(
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

		tutilaction.TC("full liquidation").
			Given(
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.InsertPosition(perpaction.WithTrader(alice), perpaction.WithPair(pairBtcUsdc), perpaction.WithSize(sdk.NewDec(10000)), perpaction.WithMargin(sdk.NewDec(1000)), perpaction.WithOpenNotional(sdk.NewDec(10600))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MsgServerMultiLiquidate(liquidator, false,
					perpaction.PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				tutilassert.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				tutilassert.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				tutilassert.BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				perpassert.PositionShouldNotExist(alice, pairBtcUsdc),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestFailMsgServer(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	app, ctx := testapp.NewNibiruTestAppAndContext()

	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	_, err := msgServer.MarketOrder(ctx, &types.MsgMarketOrder{
		Sender:               "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v",
		Pair:                 pair,
		Side:                 types.Direction_LONG,
		QuoteAssetAmount:     sdk.OneInt(),
		Leverage:             sdk.OneDec(),
		BaseAssetAmountLimit: sdk.ZeroInt(),
	})
	require.ErrorContains(t, err, "pair ubtc:unusd not found")

	_, err = msgServer.ClosePosition(ctx, &types.MsgClosePosition{
		Sender: "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v",
		Pair:   pair,
	})
	require.ErrorContains(t, err, "collections: not found:")

	_, err = msgServer.PartialClose(ctx, &types.MsgPartialClose{
		Sender: "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v",
		Pair:   pair,
		Size_:  sdk.OneDec(),
	})
	require.ErrorContains(t, err, "pair: ubtc:unusd: pair doesn't have live market")

	_, err = msgServer.MultiLiquidate(ctx, &types.MsgMultiLiquidate{
		Sender: "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v",
		Liquidations: []*types.MsgMultiLiquidate_Liquidation{
			{
				Pair:   pair,
				Trader: "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v",
			},
		},
	})
	require.ErrorContains(t, err, "pair: ubtc:unusd: pair doesn't have live market")

	_, err = msgServer.DonateToEcosystemFund(ctx, &types.MsgDonateToEcosystemFund{
		Sender:   "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v",
		Donation: sdk.NewCoin("luna", sdk.OneInt()),
	})
	require.ErrorContains(t, err, "spendable balance  is smaller than 1luna")
}

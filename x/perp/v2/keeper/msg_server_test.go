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
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestMsgServerMarketOrder(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("open long position").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
			).
			When(
				MsgServerMarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroInt()),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.999999999999"),
						Margin:                          sdk.OneDec(),
						OpenNotional:                    sdk.OneDec(),
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
				MsgServerMarketOrder(alice, pair, types.Direction_SHORT, sdk.OneInt(), sdk.OneDec(), sdk.ZeroInt()),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("-1.000000000001"),
						Margin:                          sdk.OneDec(),
						OpenNotional:                    sdk.OneDec(),
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
				MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
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
				MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
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
				MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerAddMargin(alice, pair, sdk.OneInt()),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.999999999999"),
						Margin:                          sdk.NewDec(2),
						OpenNotional:                    sdk.OneDec(),
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
				MarketOrder(alice, pair, types.Direction_LONG, sdk.NewInt(2), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerRemoveMargin(alice, pair, sdk.OneInt()),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("1.999999999996"),
						Margin:                          sdk.OneDec(),
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
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(50)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
			).
			When(
				MoveToNextBlock(),
				MsgServerMultiLiquidate(liquidator, false,
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
				MsgServerMultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(600)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(150)),
				BalanceEqual(liquidator, denoms.USDC, sdk.NewInt(250)),
				PositionShouldNotExist(alice, pairBtcUsdc),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestFailMsgServer(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	app, ctx := testapp.NewNibiruTestAppAndContext(true)

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

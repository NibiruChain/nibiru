package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"

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
	sudoerTypes "github.com/NibiruChain/nibiru/x/sudo/types"
)

func TestMsgServerMarketOrder(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("open long position").
			Given(
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
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
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(99)),
			),

		TC("open short position").
			Given(
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
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
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(99)),
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
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
				MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerClosePosition(alice, pair),
			).
			Then(
				PositionShouldNotExist(alice, pair, 1),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(100)),
			),

		TC("close short position").
			Given(
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
				MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerClosePosition(alice, pair),
			).
			Then(
				PositionShouldNotExist(alice, pair, 1),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(100)),
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
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
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
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(98)),
			),
		TC("msg server close").
			Given(
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
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
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(98)),
			),
		TC("partial close").
			Given(
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
				MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerPartialClosePosition(alice, pair, sdk.MustNewDecFromStr("0.5")),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.499999999999"),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.MustNewDecFromStr("0.499999999999250000"),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					}),
				),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(99)),
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
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
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
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(99)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerDonateToPerpEf(t *testing.T) {
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("success").
			Given(
				SetCollateral(types.TestingCollateralDenomNUSD),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
			).
			When(
				MsgServerDonateToPerpEf(alice, sdk.NewInt(50)),
			).
			Then(
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(50)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(50)),
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
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10400))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1000))),
			).
			When(
				MoveToNextBlock(),
				MsgServerMultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(750)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(125)),
				BalanceEqual(liquidator, types.TestingCollateralDenomNUSD, sdk.NewInt(125)),
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
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				InsertPosition(WithTrader(alice), WithPair(pairBtcUsdc), WithSize(sdk.NewDec(10000)), WithMargin(sdk.NewDec(1000)), WithOpenNotional(sdk.NewDec(10600))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1000))),
			).
			When(
				MoveToNextBlock(),
				MsgServerMultiLiquidate(liquidator, false,
					PairTraderTuple{Pair: pairBtcUsdc, Trader: alice, Successful: true},
				),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(600)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(150)),
				BalanceEqual(liquidator, types.TestingCollateralDenomNUSD, sdk.NewInt(250)),
				PositionShouldNotExist(alice, pairBtcUsdc, 1),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestFailMsgServer(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	app, ctx := testapp.NewNibiruTestAppAndContext()

	sender := testutil.AccAddress().String()

	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	_, err := msgServer.MarketOrder(ctx, &types.MsgMarketOrder{
		Sender:               sender,
		Pair:                 pair,
		Side:                 types.Direction_LONG,
		QuoteAssetAmount:     sdk.OneInt(),
		Leverage:             sdk.OneDec(),
		BaseAssetAmountLimit: sdk.ZeroInt(),
	})
	require.ErrorContains(t, err, "pair ubtc:unusd not found")

	_, err = msgServer.ClosePosition(ctx, &types.MsgClosePosition{
		Sender: sender,
		Pair:   pair,
	})
	require.ErrorContains(t, err, types.ErrPairNotFound.Error())

	_, err = msgServer.PartialClose(ctx, &types.MsgPartialClose{
		Sender: sender,
		Pair:   pair,
		Size_:  sdk.OneDec(),
	})
	require.ErrorContains(t, err, types.ErrPairNotFound.Error())

	_, err = msgServer.MultiLiquidate(ctx, &types.MsgMultiLiquidate{
		Sender: sender,
		Liquidations: []*types.MsgMultiLiquidate_Liquidation{
			{
				Pair:   pair,
				Trader: sender,
			},
		},
	})
	require.ErrorContains(t, err, types.ErrPairNotFound.Error())

	_, err = msgServer.DonateToEcosystemFund(ctx, &types.MsgDonateToEcosystemFund{
		Sender:   sender,
		Donation: sdk.NewCoin("luna", sdk.OneInt()),
	})
	require.ErrorContains(t, err, "spendable balance  is smaller than 1luna")
}

func TestMsgChangeCollateralDenom(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	sender := testutil.AccAddress().String()

	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	_, err := msgServer.ChangeCollateralDenom(ctx, nil)
	require.ErrorContains(t, err, "nil msg")

	_, err = msgServer.ChangeCollateralDenom(ctx, &types.MsgChangeCollateralDenom{
		Sender:   sender,
		NewDenom: "luna",
	})
	require.ErrorContains(t, err, "insufficient permissions on smart contract")

	app.SudoKeeper.Sudoers.Set(ctx, sudoerTypes.Sudoers{Contracts: []string{sender}})
	_, err = msgServer.ChangeCollateralDenom(ctx, &types.MsgChangeCollateralDenom{
		Sender:   sender,
		NewDenom: "luna",
	})
	require.NoError(t, err)

	app.SudoKeeper.Sudoers.Set(ctx, sudoerTypes.Sudoers{Contracts: []string{sender}})
	_, err = msgServer.ChangeCollateralDenom(ctx, &types.MsgChangeCollateralDenom{
		Sender:   sender,
		NewDenom: "",
	})
	require.ErrorContains(t, err, "invalid denom")

	app.SudoKeeper.Sudoers.Set(ctx, sudoerTypes.Sudoers{Contracts: []string{sender}})
	_, err = msgServer.ChangeCollateralDenom(ctx, &types.MsgChangeCollateralDenom{
		NewDenom: "luna",
	})
	require.ErrorContains(t, err, "invalid sender address")
}

func TestMsgServerSettlePosition(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("Settleposition").
			Given(
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
				MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
				CloseMarket(pair),
			).
			When(
				MsgServerSettlePosition(alice, pair, 1),
			).
			Then(
				PositionShouldNotExist(alice, pair, 1),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(100)),
			),
		TC("SettlepositionOpenedMarket").
			Given(
				CreateCustomMarket(pair, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100))),
				MarketOrder(alice, pair, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerSettlePositionShouldFail(alice, pair, 1),
			).
			Then(
				PositionShouldExist(alice, pair, 1),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestAllocateEpochRebates(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	sender := testutil.AccAddress().String()

	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	_, err := msgServer.AllocateEpochRebates(ctx, nil)
	require.ErrorContains(t, err, "nil msg")

	_, err = msgServer.AllocateEpochRebates(ctx, &types.MsgAllocateEpochRebates{})
	require.ErrorContains(t, err, "empty address string is not allowed")

	_, err = msgServer.AllocateEpochRebates(ctx, &types.MsgAllocateEpochRebates{
		Sender:  sender,
		Rebates: sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(100))),
	})
	require.ErrorContains(t, err, "insufficient funds")

	require.NoError(t,
		app.BankKeeper.MintCoins(ctx,
			inflationtypes.ModuleName,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(100))),
		),
	)
	require.NoError(t,
		app.BankKeeper.SendCoinsFromModuleToAccount(ctx,
			inflationtypes.ModuleName, sdk.MustAccAddressFromBech32(sender),
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(100)))),
	)

	_, err = msgServer.AllocateEpochRebates(ctx, &types.MsgAllocateEpochRebates{
		Sender:  sender,
		Rebates: sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(100))),
	})
	require.NoError(t, err)

	// Withdraw rebates
	_, err = msgServer.WithdrawEpochRebates(ctx, nil)
	require.ErrorContains(t, err, "nil msg")

	_, err = msgServer.WithdrawEpochRebates(ctx, &types.MsgWithdrawEpochRebates{})
	require.ErrorContains(t, err, "empty address string is not allowed")

	_, err = msgServer.WithdrawEpochRebates(ctx, &types.MsgWithdrawEpochRebates{
		Sender: sender,
		Epochs: []uint64{1},
	})
	require.ErrorContains(t, err, "collections: not found")

	currentEpoch, err := app.PerpKeeperV2.DnREpoch.Get(ctx)
	require.NoError(t, err)

	require.NoError(t, app.PerpKeeperV2.StartNewEpoch(ctx, currentEpoch+1))
	require.NoError(t, app.PerpKeeperV2.StartNewEpoch(ctx, currentEpoch+2))

	_, err = msgServer.WithdrawEpochRebates(ctx, &types.MsgWithdrawEpochRebates{
		Sender: sender,
		Epochs: []uint64{1},
	})
	require.NoError(t, err)
}

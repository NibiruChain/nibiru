package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	tutilaction "github.com/NibiruChain/nibiru/x/common/testutil/action"
	perpaction "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestQueryPositions(t *testing.T) {
	alice := testutil.AccAddress()
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	pair2 := asset.Registry.Pair(denoms.ETH, denoms.NUSD)

	tc := tutilaction.TestCases{
		tutilaction.TC("positive PnL").
			Given(
				perpaction.CreateCustomMarket(
					pair,
					perpaction.WithPricePeg(sdk.NewDec(2)),
				),
				perpaction.CreateCustomMarket(
					pair2,
					perpaction.WithPricePeg(sdk.NewDec(3)),
				),
			).
			When(
				perpaction.InsertPosition(
					perpaction.WithPair(pair),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
				perpaction.InsertPosition(
					perpaction.WithPair(pair2),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				perpaction.QueryPositions(alice,
					[]perpaction.QueryPositionChecker{
						perpaction.QueryPosition_PositionEquals(types.Position{
							Pair:                            pair,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.OneDec(),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						perpaction.QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("19.9999999998")),
						perpaction.QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("9.9999999998")),
						perpaction.QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.5499999999955")),
					},
					[]perpaction.QueryPositionChecker{
						perpaction.QueryPosition_PositionEquals(types.Position{
							Pair:                            pair2,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.OneDec(),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						perpaction.QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("29.9999999997")),
						perpaction.QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("19.9999999997")),
						perpaction.QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.699999999997")),
					},
				),
			),

		tutilaction.TC("negative PnL, positive margin ratio").
			Given(
				perpaction.CreateCustomMarket(
					pair,
					perpaction.WithPricePeg(sdk.OneDec()),
				),
				perpaction.CreateCustomMarket(
					pair2,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.95")),
				),
			).
			When(
				perpaction.InsertPosition(
					perpaction.WithPair(pair),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
				perpaction.InsertPosition(
					perpaction.WithPair(pair2),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				perpaction.QueryPositions(alice,
					[]perpaction.QueryPositionChecker{
						perpaction.QueryPosition_PositionEquals(types.Position{
							Pair:                            pair,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.OneDec(),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						perpaction.QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("9.9999999999")),
						perpaction.QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-0.0000000001")),
						perpaction.QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.099999999991")),
					},
					[]perpaction.QueryPositionChecker{
						perpaction.QueryPosition_PositionEquals(types.Position{
							Pair:                            pair2,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.OneDec(),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						perpaction.QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("9.499999999905")),
						perpaction.QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-0.500000000095")),
						perpaction.QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.052631578937894737")),
					},
				),
			),

		tutilaction.TC("negative PnL, negative margin ratio").
			Given(
				perpaction.CreateCustomMarket(
					pair,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.5")),
				),
				perpaction.CreateCustomMarket(
					pair2,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.9")),
				),
			).
			When(
				perpaction.InsertPosition(
					perpaction.WithPair(pair),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
				perpaction.InsertPosition(
					perpaction.WithPair(pair2),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				perpaction.QueryPositions(alice,
					[]perpaction.QueryPositionChecker{
						perpaction.QueryPosition_PositionEquals(types.Position{
							Pair:                            pair,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.OneDec(),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						perpaction.QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("4.99999999995")),
						perpaction.QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-5.00000000005")),
						perpaction.QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("-0.800000000018")),
					},
					[]perpaction.QueryPositionChecker{
						perpaction.QueryPosition_PositionEquals(types.Position{
							Pair:                            pair2,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.OneDec(),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						perpaction.QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("8.99999999991")),
						perpaction.QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-1.00000000009")),
						perpaction.QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("-0.00000000001")),
					},
				),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestQueryPosition(t *testing.T) {
	alice := testutil.AccAddress()
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tc := tutilaction.TestCases{
		tutilaction.TC("positive PnL").
			Given(
				perpaction.CreateCustomMarket(
					pair,
					perpaction.WithPricePeg(sdk.NewDec(2)),
				),
			).
			When(
				perpaction.InsertPosition(
					perpaction.WithPair(pair),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				perpaction.QueryPosition(pair, alice,
					perpaction.QueryPosition_PositionEquals(types.Position{
						Pair:                            pair,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.NewDec(10),
						Margin:                          sdk.OneDec(),
						OpenNotional:                    sdk.NewDec(10),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          0,
					}),
					perpaction.QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("19.9999999998")),
					perpaction.QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("9.9999999998")),
					perpaction.QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.5499999999955")),
				),
			),

		tutilaction.TC("negative PnL, positive margin ratio").
			Given(
				perpaction.CreateCustomMarket(
					pair,
					perpaction.WithPricePeg(sdk.OneDec()),
				),
			).
			When(
				perpaction.InsertPosition(
					perpaction.WithPair(pair),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				perpaction.QueryPosition(pair, alice,
					perpaction.QueryPosition_PositionEquals(types.Position{
						Pair:                            pair,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.NewDec(10),
						Margin:                          sdk.OneDec(),
						OpenNotional:                    sdk.NewDec(10),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          0,
					}),
					perpaction.QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("9.9999999999")),
					perpaction.QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-0.0000000001")),
					perpaction.QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.099999999991")),
				),
			),

		tutilaction.TC("negative PnL, negative margin ratio").
			Given(
				perpaction.CreateCustomMarket(
					pair,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.5")),
				),
			).
			When(
				perpaction.InsertPosition(
					perpaction.WithPair(pair),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				perpaction.QueryPosition(pair, alice,
					perpaction.QueryPosition_PositionEquals(types.Position{
						Pair:                            pair,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.NewDec(10),
						Margin:                          sdk.OneDec(),
						OpenNotional:                    sdk.NewDec(10),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          0,
					}),
					perpaction.QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("4.99999999995")),
					perpaction.QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-5.00000000005")),
					perpaction.QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("-0.800000000018")),
				),
			),

		tutilaction.TC("non existent position").
			Given(
				perpaction.CreateCustomMarket(
					pair,
					perpaction.WithPricePeg(sdk.NewDec(2)),
				),
			).
			When().
			Then(
				perpaction.QueryPositionNotFound(pair, alice),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestQueryMarkets(t *testing.T) {
	alice := testutil.AccAddress()
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tc := tutilaction.TestCases{
		tutilaction.TC("positive PnL").
			Given(
				perpaction.CreateCustomMarket(
					pair,
					perpaction.WithPricePeg(sdk.NewDec(2)),
				),
				tutilaction.FundModule(
					"perp_ef", sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10))),
				),
			).
			When(
				perpaction.InsertPosition(
					perpaction.WithPair(pair),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.OneDec()),
					perpaction.WithSize(sdk.NewDec(10)),
					perpaction.WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				perpaction.QueryMarkets(perpaction.QueryMarkets_MarketsShouldContain(*types.DefaultMarket(pair))),
				perpaction.QueryModuleAccounts(perpaction.QueryModuleAccounts_ModulesBalanceShouldBe(
					map[string]sdk.Coins{
						"perp_ef": sdk.NewCoins(
							sdk.NewCoin(denoms.BTC, sdk.ZeroInt()),
							sdk.NewCoin(denoms.NUSD, sdk.NewInt(10)),
						),
					},
				)),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
}

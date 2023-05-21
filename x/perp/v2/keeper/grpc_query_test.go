package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestQueryPositions(t *testing.T) {
	alice := testutil.AccAddress()
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	pair2 := asset.Registry.Pair(denoms.ETH, denoms.NUSD)

	tc := TestCases{
		TC("positive PnL").
			Given(
				CreateCustomMarket(
					pair,
					WithPricePeg(sdk.NewDec(2)),
				),
				CreateCustomMarket(
					pair2,
					WithPricePeg(sdk.NewDec(3)),
				),
			).
			When(
				InsertPosition(
					WithPair(pair),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1)),
					WithSize(sdk.NewDec(10)),
					WithOpenNotional(sdk.NewDec(10)),
				),
				InsertPosition(
					WithPair(pair2),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1)),
					WithSize(sdk.NewDec(10)),
					WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				QueryPositions(alice,
					[]QueryPositionChecker{
						QueryPosition_PositionEquals(types.Position{
							Pair:                            pair,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.NewDec(1),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("19.9999999998")),
						QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("9.9999999998")),
						QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.5499999999955")),
					},
					[]QueryPositionChecker{
						QueryPosition_PositionEquals(types.Position{
							Pair:                            pair2,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.NewDec(1),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("29.9999999997")),
						QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("19.9999999997")),
						QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.699999999997")),
					},
				),
			),

		TC("negative PnL, positive margin ratio").
			Given(
				CreateCustomMarket(
					pair,
					WithPricePeg(sdk.NewDec(1)),
				),
				CreateCustomMarket(
					pair2,
					WithPricePeg(sdk.MustNewDecFromStr("0.95")),
				),
			).
			When(
				InsertPosition(
					WithPair(pair),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1)),
					WithSize(sdk.NewDec(10)),
					WithOpenNotional(sdk.NewDec(10)),
				),
				InsertPosition(
					WithPair(pair2),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1)),
					WithSize(sdk.NewDec(10)),
					WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				QueryPositions(alice,
					[]QueryPositionChecker{
						QueryPosition_PositionEquals(types.Position{
							Pair:                            pair,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.NewDec(1),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("9.9999999999")),
						QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-0.0000000001")),
						QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.099999999991")),
					},
					[]QueryPositionChecker{
						QueryPosition_PositionEquals(types.Position{
							Pair:                            pair2,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.NewDec(1),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("9.499999999905")),
						QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-0.500000000095")),
						QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.052631578937894737")),
					},
				),
			),

		TC("negative PnL, negative margin ratio").
			Given(
				CreateCustomMarket(
					pair,
					WithPricePeg(sdk.MustNewDecFromStr("0.5")),
				),
				CreateCustomMarket(
					pair2,
					WithPricePeg(sdk.MustNewDecFromStr("0.9")),
				),
			).
			When(
				InsertPosition(
					WithPair(pair),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1)),
					WithSize(sdk.NewDec(10)),
					WithOpenNotional(sdk.NewDec(10)),
				),
				InsertPosition(
					WithPair(pair2),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1)),
					WithSize(sdk.NewDec(10)),
					WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				QueryPositions(alice,
					[]QueryPositionChecker{
						QueryPosition_PositionEquals(types.Position{
							Pair:                            pair,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.NewDec(1),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("4.99999999995")),
						QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-5.00000000005")),
						QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("-0.800000000018")),
					},
					[]QueryPositionChecker{
						QueryPosition_PositionEquals(types.Position{
							Pair:                            pair2,
							TraderAddress:                   alice.String(),
							Size_:                           sdk.NewDec(10),
							Margin:                          sdk.NewDec(1),
							OpenNotional:                    sdk.NewDec(10),
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
							LastUpdatedBlockNumber:          0,
						}),
						QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("8.99999999991")),
						QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-1.00000000009")),
						QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("-0.00000000001")),
					},
				),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestQueryPosition(t *testing.T) {
	alice := testutil.AccAddress()
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tc := TestCases{
		TC("positive PnL").
			Given(
				CreateCustomMarket(
					pair,
					WithPricePeg(sdk.NewDec(2)),
				),
			).
			When(
				InsertPosition(
					WithPair(pair),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1)),
					WithSize(sdk.NewDec(10)),
					WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				QueryPosition(pair, alice,
					QueryPosition_PositionEquals(types.Position{
						Pair:                            pair,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.NewDec(10),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.NewDec(10),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          0,
					}),
					QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("19.9999999998")),
					QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("9.9999999998")),
					QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.5499999999955")),
				),
			),

		TC("negative PnL, positive margin ratio").
			Given(
				CreateCustomMarket(
					pair,
					WithPricePeg(sdk.NewDec(1)),
				),
			).
			When(
				InsertPosition(
					WithPair(pair),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1)),
					WithSize(sdk.NewDec(10)),
					WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				QueryPosition(pair, alice,
					QueryPosition_PositionEquals(types.Position{
						Pair:                            pair,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.NewDec(10),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.NewDec(10),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          0,
					}),
					QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("9.9999999999")),
					QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-0.0000000001")),
					QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("0.099999999991")),
				),
			),

		TC("negative PnL, negative margin ratio").
			Given(
				CreateCustomMarket(
					pair,
					WithPricePeg(sdk.MustNewDecFromStr("0.5")),
				),
			).
			When(
				InsertPosition(
					WithPair(pair),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1)),
					WithSize(sdk.NewDec(10)),
					WithOpenNotional(sdk.NewDec(10)),
				),
			).
			Then(
				QueryPosition(pair, alice,
					QueryPosition_PositionEquals(types.Position{
						Pair:                            pair,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.NewDec(10),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.NewDec(10),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          0,
					}),
					QueryPosition_PositionNotionalEquals(sdk.MustNewDecFromStr("4.99999999995")),
					QueryPosition_UnrealizedPnlEquals(sdk.MustNewDecFromStr("-5.00000000005")),
					QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("-0.800000000018")),
				),
			),

		TC("non existent position").
			Given(
				CreateCustomMarket(
					pair,
					WithPricePeg(sdk.NewDec(2)),
				),
			).
			When().
			Then(
				QueryPositionNotFound(pair, alice),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

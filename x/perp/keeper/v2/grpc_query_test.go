package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	keeper "github.com/NibiruChain/nibiru/x/perp/keeper/v2"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func initAppMarkets(
	t *testing.T, sqrtReserve, priceMultiplier sdk.Dec,
) (sdk.Context, *app.NibiruApp, v2types.QueryServer) {
	t.Log("initialize app and keeper")
	app, ctx := testapp.NewNibiruTestAppAndContext(true)
	queryServer := keeper.NewQuerier(app.PerpKeeperV2)

	t.Log("initialize market and pair")
	market := mock.TestMarket()
	amm := mock.TestAMM(sqrtReserve, priceMultiplier)
	app.PerpKeeperV2.Markets.Insert(ctx, market.Pair, *market)
	app.PerpKeeperV2.AMMs.Insert(ctx, amm.Pair, *amm)

	market = mock.TestMarket().WithPair(asset.Registry.Pair(denoms.ETH, denoms.NUSD))
	amm = mock.TestAMM(sqrtReserve, priceMultiplier).WithPair(asset.Registry.Pair(denoms.ETH, denoms.NUSD))
	app.PerpKeeperV2.Markets.Insert(ctx, market.Pair, *market)
	app.PerpKeeperV2.AMMs.Insert(ctx, amm.Pair, *amm)

	return ctx, app, queryServer
}

func TestQueryPositions(t *testing.T) {
	tests := []struct {
		name      string
		Positions []*v2types.Position
	}{
		{
			name: "positive PnL",
			Positions: []*v2types.Position{
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					Size_:                           sdk.NewDec(10),
					OpenNotional:                    sdk.NewDec(10),
					Margin:                          sdk.NewDec(1),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					Pair:                            asset.Registry.Pair(denoms.ETH, denoms.NUSD),
					Size_:                           sdk.NewDec(10),
					OpenNotional:                    sdk.NewDec(10),
					Margin:                          sdk.NewDec(1),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("initialize trader address")
			traderAddr := testutil.AccAddress()

			tc.Positions[0].TraderAddress = traderAddr.String()
			tc.Positions[1].TraderAddress = traderAddr.String()

			ctx, app, queryServer := initAppMarkets(
				t,
				sdk.NewDec(100_000),
				sdk.OneDec(),
			)

			t.Log("initialize position")
			for _, position := range tc.Positions {
				currentPosition := position
				currentPosition.TraderAddress = traderAddr.String()
				app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(currentPosition.Pair, traderAddr), *currentPosition)
			}

			t.Log("query position")
			ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second))
			resp, err := queryServer.QueryPositions(
				sdk.WrapSDKContext(ctx),
				&v2types.QueryPositionsRequest{
					Trader: traderAddr.String(),
				},
			)
			require.NoError(t, err)

			t.Log("assert response")
			assert.Equal(t, len(tc.Positions), len(resp.Positions))
		})
	}
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
					QueryPosition_PositionEquals(v2types.Position{
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
					QueryPosition_PositionEquals(v2types.Position{
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
					QueryPosition_PositionEquals(v2types.Position{
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
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

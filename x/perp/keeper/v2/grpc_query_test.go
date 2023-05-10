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
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
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

func TestQueryPosition(t *testing.T) {
	tests := []struct {
		name            string
		initialPosition *v2types.Position

		sqrtDepth       sdk.Dec
		priceMultiplier sdk.Dec

		expectedPositionNotional sdk.Dec
		expectedUnrealizedPnl    sdk.Dec
		expectedMarginRatio      sdk.Dec
	}{
		{
			name: "positive PnL",
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				LastUpdatedBlockNumber:          1,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			sqrtDepth:       sdk.NewDec(1e6),
			priceMultiplier: sdk.NewDec(2),

			expectedPositionNotional: sdk.MustNewDecFromStr("19.999800001999980000"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("9.999800001999980000"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("0.549995500000000000"),
		},
		{
			name: "negative PnL, positive margin ratio",
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				LastUpdatedBlockNumber:          1,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			sqrtDepth:       sdk.NewDec(1e6),
			priceMultiplier: sdk.OneDec(),

			expectedPositionNotional: sdk.MustNewDecFromStr("9.99990000099999"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-0.00009999900001"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("0.099991"),
		},
		{
			name: "negative PnL, negative margin ratio",
			initialPosition: &v2types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				LastUpdatedBlockNumber:          1,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			sqrtDepth:       sdk.NewDec(1e6),
			priceMultiplier: sdk.MustNewDecFromStr("0.5"),

			expectedPositionNotional: sdk.MustNewDecFromStr("4.999950000499995"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-5.000049999500005"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("-0.800018"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("initialize trader address")
			traderAddr := testutil.AccAddress()
			tc.initialPosition.TraderAddress = traderAddr.String()

			t.Log("initialize app and keeper")
			ctx, app, queryServer := initAppMarkets(t, tc.sqrtDepth, tc.priceMultiplier)

			t.Log("initialize position")
			app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(tc.initialPosition.Pair, traderAddr), *tc.initialPosition)

			t.Log("query position")
			ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second))
			resp, err := queryServer.QueryPosition(
				sdk.WrapSDKContext(ctx),
				&v2types.QueryPositionRequest{
					Trader: traderAddr.String(),
					Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				},
			)
			require.NoError(t, err)

			t.Log("assert response")
			assert.EqualValues(t, tc.initialPosition, resp.Position)

			assert.Equal(t, tc.expectedPositionNotional, resp.PositionNotional)
			assert.Equal(t, tc.expectedUnrealizedPnl, resp.UnrealizedPnl)
			assert.Equal(t, tc.expectedMarginRatio, resp.MarginRatio)
		})
	}
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

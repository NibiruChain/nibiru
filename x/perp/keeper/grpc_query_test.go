package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func initAppVpools(
	t *testing.T, quoteAssetReserve sdk.Dec, baseAssetReserve sdk.Dec,
) (sdk.Context, *app.NibiruApp, types.QueryServer) {
	t.Log("initialize app and keeper")
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
	perpKeeper := &nibiruApp.PerpKeeper
	vpoolKeeper := &nibiruApp.VpoolKeeper
	queryServer := keeper.NewQuerier(*perpKeeper)

	t.Log("initialize vpool and pair")
	assert.NoError(t, vpoolKeeper.CreatePool(
		ctx,
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		quoteAssetReserve,
		baseAssetReserve,
		perpammtypes.VpoolConfig{
			TradeLimitRatio:        sdk.OneDec(),
			FluctuationLimitRatio:  sdk.OneDec(),
			MaxOracleSpreadRatio:   sdk.OneDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
		sdk.ZeroDec(),
		sdk.OneDec(),
	))
	keeper.SetPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
		Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
	})
	assert.NoError(t, vpoolKeeper.CreatePool(
		ctx,
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		/* quoteReserve */ sdk.MustNewDecFromStr("100000"),
		/* baseReserve */ sdk.MustNewDecFromStr("100000"),
		perpammtypes.VpoolConfig{
			TradeLimitRatio:        sdk.OneDec(),
			FluctuationLimitRatio:  sdk.OneDec(),
			MaxOracleSpreadRatio:   sdk.OneDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
		sdk.ZeroDec(),
		sdk.OneDec(),
	))
	keeper.SetPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
		Pair:                            asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
	})
	assert.NoError(t, vpoolKeeper.CreatePool(
		ctx,
		asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
		/* quoteReserve */ sdk.MustNewDecFromStr("100000"),
		/* baseReserve */ sdk.MustNewDecFromStr("100000"),
		perpammtypes.VpoolConfig{
			TradeLimitRatio:        sdk.OneDec(),
			FluctuationLimitRatio:  sdk.OneDec(),
			MaxOracleSpreadRatio:   sdk.OneDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
		sdk.ZeroDec(),
		sdk.OneDec(),
	))
	keeper.SetPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
		Pair:                            asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
	})
	return ctx, nibiruApp, queryServer
}

func TestQueryPosition(t *testing.T) {
	tests := []struct {
		name            string
		initialPosition *types.Position

		quoteAssetReserve sdk.Dec
		baseAssetReserve  sdk.Dec

		expectedPositionNotional sdk.Dec
		expectedUnrealizedPnl    sdk.Dec
		expectedMarginRatio      sdk.Dec
	}{
		{
			name: "positive PnL",
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				BlockNumber:                     1,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			quoteAssetReserve: sdk.NewDec(1 * common.TO_MICRO),
			baseAssetReserve:  sdk.NewDec(500_000),

			expectedPositionNotional: sdk.MustNewDecFromStr("19.999600007999840003"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("9.999600007999840003"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("0.549991"),
		},
		{
			name: "negative PnL, positive margin ratio",
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				BlockNumber:                     1,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			quoteAssetReserve: sdk.NewDec(1 * common.TO_MICRO),
			baseAssetReserve:  sdk.NewDec(1 * common.TO_MICRO),

			expectedPositionNotional: sdk.MustNewDecFromStr("9.99990000099999"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-0.00009999900001"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("0.099991"),
		},
		{
			name: "negative PnL, negative margin ratio",
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				BlockNumber:                     1,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			quoteAssetReserve: sdk.NewDec(500_000),
			baseAssetReserve:  sdk.NewDec(1 * common.TO_MICRO),

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
			ctx, app, queryServer := initAppVpools(t, tc.quoteAssetReserve, tc.baseAssetReserve)

			t.Log("initialize position")
			keeper.SetPosition(app.PerpKeeper, ctx, *tc.initialPosition)

			t.Log("query position")
			ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second))
			resp, err := queryServer.QueryPosition(
				sdk.WrapSDKContext(ctx),
				&types.QueryPositionRequest{
					Trader: traderAddr.String(),
					Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				},
			)
			require.NoError(t, err)

			t.Log("assert response")
			assert.EqualValues(t, tc.initialPosition, resp.Position)

			assert.Equal(t, tc.expectedPositionNotional, resp.PositionNotional)
			assert.Equal(t, tc.expectedUnrealizedPnl, resp.UnrealizedPnl)
			assert.Equal(t, tc.expectedMarginRatio, resp.MarginRatioMark)
			// assert.Equal(t, tc.expectedMarginRatioIndex, resp.MarginRatioIndex)
			// TODO https://github.com/NibiruChain/nibiru/issues/809
		})
	}
}

func TestQueryPositions(t *testing.T) {
	tests := []struct {
		name      string
		Positions []*types.Position
	}{
		{
			name: "positive PnL",
			Positions: []*types.Position{
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					Size_:                           sdk.NewDec(10),
					OpenNotional:                    sdk.NewDec(10),
					Margin:                          sdk.NewDec(1),
					BlockNumber:                     1,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					Pair:                            asset.Registry.Pair(denoms.ETH, denoms.NUSD),
					Size_:                           sdk.NewDec(10),
					OpenNotional:                    sdk.NewDec(10),
					Margin:                          sdk.NewDec(1),
					BlockNumber:                     1,
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
			tc.Positions[0].TraderAddress = traderAddr.String()

			ctx, app, queryServer := initAppVpools(
				t,
				/* quoteReserve */ sdk.NewDec(100_000),
				/* baseReserve */ sdk.NewDec(100_000),
			)

			t.Log("initialize position")
			for _, position := range tc.Positions {
				currentPosition := position
				currentPosition.TraderAddress = traderAddr.String()
				keeper.SetPosition(app.PerpKeeper, ctx, *currentPosition)
			}

			t.Log("query position")
			ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second))
			resp, err := queryServer.QueryPositions(
				sdk.WrapSDKContext(ctx),
				&types.QueryPositionsRequest{
					Trader: traderAddr.String(),
				},
			)
			require.NoError(t, err)

			t.Log("assert response")
			assert.Equal(t, len(tc.Positions), len(resp.Positions))
		})
	}
}

func TestQueryCumulativePremiumFraction(t *testing.T) {
	tests := []struct {
		name                string
		initialPairMetadata *types.PairMetadata

		query *types.QueryCumulativePremiumFractionRequest

		expectErr            bool
		expectedLatestCPF    sdk.Dec
		expectedEstimatedCPF sdk.Dec
	}{
		{
			name: "empty string pair",
			initialPairMetadata: &types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			query: &types.QueryCumulativePremiumFractionRequest{
				Pair: "",
			},
			expectErr: true,
		},
		{
			name: "pair metadata not found",
			initialPairMetadata: &types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			query: &types.QueryCumulativePremiumFractionRequest{
				Pair: "foo:bar",
			},
			expectErr: true,
		},
		{
			name: "returns single funding payment",
			initialPairMetadata: &types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			query: &types.QueryCumulativePremiumFractionRequest{
				Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			},
			expectErr:            false,
			expectedLatestCPF:    sdk.ZeroDec(),
			expectedEstimatedCPF: sdk.NewDec(10), // (481 - 1) / 48
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("initialize app and keeper")
			ctx, app, queryServer := initAppVpools(t, sdk.NewDec(481_000), sdk.NewDec(1_000))

			t.Log("set index price")
			app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), sdk.OneDec())

			t.Log("query cumulative premium fraction")
			resp, err := queryServer.CumulativePremiumFraction(sdk.WrapSDKContext(ctx), tc.query)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.EqualValues(t, tc.expectedLatestCPF, resp.CumulativePremiumFraction)
				assert.EqualValues(t, tc.expectedEstimatedCPF, resp.EstimatedNextCumulativePremiumFraction)
			}
		})
	}
}

func TestQueryMetrics(t *testing.T) {
	tests := []struct {
		name        string
		Positions   []*types.Position
		NetSize     sdk.Dec
		VolumeBase  sdk.Dec
		VolumeQuote sdk.Dec
	}{
		{
			name:        "no positions",
			Positions:   []*types.Position{},
			NetSize:     sdk.ZeroDec(),
			VolumeBase:  sdk.ZeroDec(),
			VolumeQuote: sdk.ZeroDec(),
		},
		{
			name: "two longs",
			Positions: []*types.Position{
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "WHALE",
					Size_:                           sdk.NewDec(10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					TraderAddress:                   "SHRIMP",
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					Size_:                           sdk.NewDec(10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
			},
			NetSize:     sdk.NewDec(20),
			VolumeBase:  sdk.NewDec(20),
			VolumeQuote: sdk.NewDec(200),
		},
		{
			name: "two shorts",
			Positions: []*types.Position{
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "WHALE",
					Size_:                           sdk.NewDec(-10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					TraderAddress:                   "SHRIMP",
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					Size_:                           sdk.NewDec(-10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
			},
			NetSize:     sdk.NewDec(-20),
			VolumeBase:  sdk.NewDec(20),
			VolumeQuote: sdk.NewDec(200),
		},
		{
			name: "one long, one short",
			Positions: []*types.Position{
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "WHALE",
					Size_:                           sdk.NewDec(10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					TraderAddress:                   "SHRIMP",
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					Size_:                           sdk.NewDec(-10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
			},
			NetSize:     sdk.NewDec(0),
			VolumeBase:  sdk.NewDec(20),
			VolumeQuote: sdk.NewDec(200),
		},
		{
			name: "decrease position",
			Positions: []*types.Position{
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "WHALE",
					Size_:                           sdk.NewDec(10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "WHALE",
					Size_:                           sdk.NewDec(-10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
			},
			NetSize:     sdk.NewDec(0),
			VolumeBase:  sdk.NewDec(20),
			VolumeQuote: sdk.NewDec(200),
		},
		{
			name: "swap positions",
			Positions: []*types.Position{
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "WHALE",
					Size_:                           sdk.NewDec(10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "SHRIMP",
					Size_:                           sdk.NewDec(-10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "WHALE",
					Size_:                           sdk.NewDec(-10),
					OpenNotional:                    sdk.NewDec(100),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "WHALE",
					Size_:                           sdk.NewDec(-20),
					OpenNotional:                    sdk.NewDec(200),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "SHRIMP",
					Size_:                           sdk.NewDec(20),
					OpenNotional:                    sdk.NewDec(200),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
				{
					Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					TraderAddress:                   "SHRIMP",
					Size_:                           sdk.NewDec(20),
					OpenNotional:                    sdk.NewDec(200),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				},
			},
			NetSize:     sdk.NewDec(10),
			VolumeBase:  sdk.NewDec(90),
			VolumeQuote: sdk.NewDec(900),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx, app, queryServer := initAppVpools(
				t,
				/* quoteReserve */ sdk.NewDec(100_000),
				/* baseReserve */ sdk.NewDec(100_000),
			)
			t.Log("call OnSwapEnd hook")
			for _, position := range tc.Positions {
				// Detect position decrease
				app.PerpKeeper.OnSwapEnd(
					ctx,
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					position.OpenNotional,
					position.Size_,
				)
			}

			t.Log("query metrics")
			resp, err := queryServer.Metrics(
				sdk.WrapSDKContext(ctx),
				&types.QueryMetricsRequest{
					Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				},
			)
			require.NoError(t, err)

			t.Log("assert response")
			assert.Equal(t, tc.NetSize, resp.Metrics.NetSize)
			assert.Equal(t, tc.VolumeQuote, resp.Metrics.VolumeQuote)
			assert.Equal(t, tc.VolumeBase, resp.Metrics.VolumeBase)
		})
	}
}

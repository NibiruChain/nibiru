package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

func TestQueryReserveAssets(t *testing.T) {
	t.Log("initialize vpoolkeeper")
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockOracleKeeper(gomock.NewController(t)),
	)
	queryServer := NewQuerier(vpoolKeeper)

	t.Log("initialize vpool")
	pool := types.Vpool{
		Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		QuoteAssetReserve: sdk.NewDec(1 * common.TO_MICRO),
		BaseAssetReserve:  sdk.NewDec(1000),
		Config: types.VpoolConfig{
			FluctuationLimitRatio:  sdk.ZeroDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.ZeroDec(),
			TradeLimitRatio:        sdk.ZeroDec(),
		},
	}
	vpoolKeeper.Pools.Insert(ctx, pool.Pair, pool)

	t.Log("query reserve assets")
	resp, err := queryServer.ReserveAssets(
		sdk.WrapSDKContext(ctx),
		&types.QueryReserveAssetsRequest{
			Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		},
	)

	t.Log("assert reserve assets")
	require.NoError(t, err)
	assert.EqualValues(t, pool.QuoteAssetReserve, resp.QuoteAssetReserve)
	assert.EqualValues(t, pool.BaseAssetReserve, resp.BaseAssetReserve)
}

func TestQueryAllPools(t *testing.T) {
	t.Log("initialize vpoolkeeper")
	vpoolKeeper, mocks, ctx := getKeeper(t)
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())
	queryServer := NewQuerier(vpoolKeeper)

	t.Log("initialize vpool")
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	pool := &types.Vpool{
		Pair:              pair,
		QuoteAssetReserve: sdk.NewDec(1 * common.TO_MICRO),
		BaseAssetReserve:  sdk.NewDec(1000),
		Config: types.VpoolConfig{
			FluctuationLimitRatio:  sdk.ZeroDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.ZeroDec(),
			TradeLimitRatio:        sdk.ZeroDec(),
		},
	}
	assert.NoError(t, vpoolKeeper.CreatePool(
		ctx, pair, pool.QuoteAssetReserve, pool.BaseAssetReserve, pool.Config, sdk.ZeroDec(), sdk.OneDec()))

	t.Log("query reserve assets and prices for the pair")
	ctx = ctx.WithBlockHeight(2).WithBlockTime(time.Now().Add(5 * time.Second))
	indexPrice := sdk.NewDec(25_000)
	mocks.mockOracleKeeper.EXPECT().
		GetExchangeRate(ctx, pair).
		Return(indexPrice, nil)
	resp, err := queryServer.AllPools(
		sdk.WrapSDKContext(ctx),
		&types.QueryAllPoolsRequest{},
	)

	t.Log("check if query response is accurate")
	markPriceWanted := sdk.NewDec(1_000) // 1e6 / 1e3
	poolPricesWanted := types.PoolPrices{
		Pair:          pool.Pair,
		MarkPrice:     markPriceWanted,
		IndexPrice:    indexPrice.String(),
		TwapMark:      markPriceWanted.String(),
		SwapInvariant: sdk.NewInt(1_000 * common.TO_MICRO),
		BlockNumber:   2,
	}
	require.NoError(t, err)
	assert.EqualValues(t, pool.Pair, resp.Pools[0].Pair)
	assert.EqualValues(t, poolPricesWanted, resp.Prices[0])
}

package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestQueryReserveAssets(t *testing.T) {
	t.Log("initialize vpoolkeeper")
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)
	queryServer := NewQuerier(vpoolKeeper)

	t.Log("initialize vpool")
	pool := &types.VPool{
		Pair:                   common.Pair_BTC_NUSD,
		TradeLimitRatio:        sdk.ZeroDec(),
		QuoteAssetReserve:      sdk.NewDec(1_000_000),
		BaseAssetReserve:       sdk.NewDec(1000),
		FluctuationLimitRatio:  sdk.ZeroDec(),
		MaxOracleSpreadRatio:   sdk.ZeroDec(),
		MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
		MaxLeverage:            sdk.MustNewDecFromStr("15"),
	}
	vpoolKeeper.savePool(ctx, pool)

	t.Log("query reserve assets")
	resp, err := queryServer.ReserveAssets(
		sdk.WrapSDKContext(ctx),
		&types.QueryReserveAssetsRequest{
			Pair: common.Pair_BTC_NUSD.String(),
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
	pair := common.Pair_BTC_NUSD
	pool := &types.VPool{
		Pair:                   pair,
		TradeLimitRatio:        sdk.ZeroDec(),
		QuoteAssetReserve:      sdk.NewDec(1_000_000),
		BaseAssetReserve:       sdk.NewDec(1000),
		FluctuationLimitRatio:  sdk.ZeroDec(),
		MaxOracleSpreadRatio:   sdk.ZeroDec(),
		MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
		MaxLeverage:            sdk.MustNewDecFromStr("15"),
	}
	vpoolKeeper.CreatePool(
		ctx, pair, pool.TradeLimitRatio, pool.QuoteAssetReserve, pool.BaseAssetReserve, pool.FluctuationLimitRatio, pool.MaxOracleSpreadRatio, pool.MaintenanceMarginRatio, pool.MaxLeverage)

	t.Log("query reserve assets and prices for the pair")
	ctx = ctx.WithBlockHeight(2).WithBlockTime(time.Now().Add(5 * time.Second))
	indexPrice := sdk.NewDec(25_000)
	mocks.mockPricefeedKeeper.EXPECT().
		GetCurrentPrice(ctx, pair.BaseDenom(), pair.QuoteDenom()).
		Return(pricefeedtypes.CurrentPrice{PairID: pair.String(), Price: indexPrice}, nil)
	resp, err := queryServer.AllPools(
		sdk.WrapSDKContext(ctx),
		&types.QueryAllPoolsRequest{},
	)

	t.Log("check if query response is accurate")
	markPriceWanted := sdk.NewDec(1_000) // 1e6 / 1e3
	poolPricesWanted := types.PoolPrices{
		Pair:          pool.Pair.String(),
		MarkPrice:     markPriceWanted,
		IndexPrice:    indexPrice.String(),
		TwapMark:      markPriceWanted.String(),
		SwapInvariant: sdk.NewInt(1_000_000_000), // 1e6 * 1e3
		BlockNumber:   2,
	}
	require.NoError(t, err)
	assert.EqualValues(t, pool.Pair, resp.Pools[0].Pair)
	assert.EqualValues(t, poolPricesWanted, resp.Prices[0])
}

package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestKeeper_SetGetPair(t *testing.T) {
	app, ctx := simapp.NewTestNibiruAppAndContext(true)

	pairs := common.AssetPairs{
		common.MustNewAssetPair("tst:usd"),
		common.MustNewAssetPair("xyz:abc"),
	}

	keeper := app.PricefeedKeeper
	keeper.SetParams(ctx, types.Params{Pairs: common.AssetPairs{pairs[0]}})

	paramsPairs := keeper.GetPairs(ctx)
	require.Len(t, paramsPairs, 1)
	require.Equal(t, paramsPairs[0].String(), "tst:usd")

	require.True(t, keeper.IsActivePair(ctx, pairs[0].String()))
	require.True(t, !keeper.IsActivePair(ctx, pairs[1].String()))

	params := types.Params{Pairs: pairs}
	keeper.SetParams(ctx, params)
	paramsPairs = keeper.GetPairs(ctx)
	require.Len(t, paramsPairs, 2)
	for _, pair := range paramsPairs {
		require.True(t, keeper.IsActivePair(ctx, pair.String()))
	}
	require.False(t, keeper.IsActivePair(ctx, "nan:nan"))
}

func TestKeeper_GetPostRawPrice(t *testing.T) {
	app, ctx := simapp.NewTestNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper

	_, addrs := sample.PrivKeyAddressPairs(2)
	pair := common.MustNewAssetPair("tst:usd")
	params := types.Params{Pairs: common.AssetPairs{pair}}
	keeper.SetParams(ctx, params)
	keeper.OraclesStore().AddOracles(ctx, pair, addrs)

	rawPrices := []struct {
		oracle sdk.AccAddress
		pair   common.AssetPair
		price  sdk.Dec
		expiry time.Time
		total  int
	}{
		{addrs[0], pair, sdk.MustNewDecFromStr("0.33"), time.Now().UTC().Add(1 * time.Hour), 1},
		{addrs[1], pair, sdk.MustNewDecFromStr("0.35"), time.Now().UTC().Add(1 * time.Hour), 2},
		{addrs[0], pair, sdk.MustNewDecFromStr("0.37"), time.Now().UTC().Add(1 * time.Hour), 2},
	}

	for _, rawPrice := range rawPrices {
		// Set price by oracle 1

		require.NoError(t, keeper.PostRawPrice(
			ctx,
			rawPrice.oracle,
			rawPrice.pair.String(),
			rawPrice.price,
			rawPrice.expiry,
		))

		// Get raw prices
		rawPrices := keeper.GetRawPrices(ctx, pair.String())

		require.Equal(t, rawPrice.total, len(rawPrices))
		require.Contains(t, rawPrices, types.PostedPrice{
			PairID: rawPrice.pair.String(),
			Oracle: rawPrice.oracle.String(),
			Price:  rawPrice.price,
			Expiry: rawPrice.expiry,
		})

		// Find the oracle and require price to be same
		for _, postedPrice := range rawPrices {
			oracleThatPosted := sdk.MustAccAddressFromBech32(postedPrice.Oracle)
			if rawPrice.oracle.Equals(oracleThatPosted) {
				require.Equal(t, rawPrice.price, postedPrice.Price)
			}
		}
	}
}

/*
Test case where two oracles try to set prices for a market and only one of the
oracles is valid (i.e. registered with keeper.SetParams).
*/
func TestKeeper_PostRawPriceWrongOracle(t *testing.T) {
	app, ctx := simapp.NewTestNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper
	pair := common.MustNewAssetPair("tst:usd")

	// Register addrs[1] as the oracle.
	_, addrs := sample.PrivKeyAddressPairs(2)

	params := types.Params{Pairs: common.AssetPairs{pair}}
	keeper.SetParams(ctx, params)

	// Set price with valid oracle given (addrs[0])
	keeper.WhitelistOracles(ctx, []sdk.AccAddress{addrs[0]})
	expiry := ctx.BlockTime().UTC().Add(1 * time.Hour)
	require.NoError(t, keeper.PostRawPrice(
		ctx, addrs[0], pair.String(), sdk.MustNewDecFromStr("0.1"), expiry,
	))

	// Set price with invalid oracle given (addrs[1])
	require.Error(t, keeper.PostRawPrice(
		ctx, addrs[1], pair.String(), sdk.MustNewDecFromStr("0.1"), expiry,
	))
}

/*
Test case where several oracles try to set prices for a market
and "k" (int) of the oracles are valid (i.e. registered with keeper.SetParams).
*/
func TestKeeper_PostRawPriceWrongOracles(t *testing.T) {
	app, ctx := simapp.NewTestNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper

	pair := common.MustNewAssetPair("tst:usd")
	price := sdk.MustNewDecFromStr("0.1")

	_, addrs := sample.PrivKeyAddressPairs(10)
	params := types.Params{
		Pairs: common.AssetPairs{pair},
	}
	keeper.SetParams(ctx, params)
	keeper.WhitelistOraclesForPairs(ctx, addrs[:5], common.AssetPairs{pair})

	for i, addr := range addrs {
		if i < 5 {
			// Valid oracle addresses. This shouldn't raise an error.
			require.NoError(t, keeper.PostRawPrice(
				ctx, addr, pair.String(), price, time.Now().UTC().Add(1*time.Hour),
			))
		} else {
			// Invalid oracle addresses. This should raise errors.
			require.Error(t, keeper.PostRawPrice(
				ctx, addr, pair.String(), price, time.Now().UTC().Add(1*time.Hour),
			))
		}
	}
}

// Test Setting the current price of an Asset
func TestKeeper_GetSetCurrentPrice(t *testing.T) {
	_, addrs := sample.PrivKeyAddressPairs(5)
	app, ctx := simapp.NewTestNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper

	pair := common.Pair_BTC_NUSD
	keeper.OraclesStore().AddOracles(ctx, pair, addrs)
	keeper.SetParams(ctx, types.Params{
		Pairs:              common.AssetPairs{pair},
		TwapLookbackWindow: 15 * time.Minute,
	})

	require.NoError(t, keeper.PostRawPrice(ctx, addrs[0], pair.String(), sdk.MustNewDecFromStr("0.33"), time.Now().Add(time.Hour)))
	require.NoError(t, keeper.PostRawPrice(ctx, addrs[1], pair.String(), sdk.MustNewDecFromStr("0.35"), time.Now().Add(time.Hour)))
	require.NoError(t, keeper.PostRawPrice(ctx, addrs[2], pair.String(), sdk.MustNewDecFromStr("0.34"), time.Now().Add(time.Hour)))

	t.Log("Add an expired one which should fail")
	require.Error(t, keeper.PostRawPrice(ctx, addrs[3], pair.String(), sdk.MustNewDecFromStr("0.9"), ctx.BlockTime().Add(-time.Hour)))

	t.Log("Add a non-expired price, but will not be counted when BlockTime is changed")
	require.NoError(t, keeper.PostRawPrice(ctx, addrs[3], pair.String(), sdk.MustNewDecFromStr("0.9"), time.Now().Add(time.Minute*30)))

	// Update block time such that first 3 prices valid but last one is expired
	ctx = ctx.WithBlockTime(time.Now().Add(time.Minute * 45)).WithBlockHeight(1)

	// Set current price
	require.NoError(t, keeper.GatherRawPrices(ctx, pair.Token0, pair.Token1))

	// Get current currentPrice
	currentPrice, err := keeper.GetCurrentPrice(ctx, pair.Token0, pair.Token1)
	require.NoError(t, err)
	assert.Equal(t, sdk.MustNewDecFromStr("0.34"), currentPrice.Price)

	// Check TWAP Price
	twap, err := keeper.GetCurrentTWAP(ctx, pair.Token0, pair.Token1)
	require.NoError(t, err)
	assert.Equal(t, sdk.MustNewDecFromStr("0.34"), twap)

	// fast forward block height as twap snapshots are indexed by blockHeight
	ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(10 * time.Second))

	require.NoError(t, keeper.PostRawPrice(ctx, addrs[4], pair.String(), sdk.MustNewDecFromStr("0.36"), time.Now().Add(time.Hour)))
	require.NoError(t, keeper.GatherRawPrices(ctx, pair.Token0, pair.Token1))

	currentPrice, err = keeper.GetCurrentPrice(ctx, pair.Token0, pair.Token1)
	require.Nil(t, err)
	require.Equal(t, sdk.MustNewDecFromStr("0.345"), currentPrice.Price)

	prices := keeper.GetCurrentPrices(ctx)
	require.Equal(t, 1, len(prices))
	require.Equal(t, currentPrice, prices[0])

	// Check TWAP Price
	ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(10 * time.Second))
	twap, err = keeper.GetCurrentTWAP(ctx, pair.Token0, pair.Token1)
	require.NoError(t, err)
	require.Equal(t, sdk.MustNewDecFromStr("0.3425"), twap)
}

func TestGetCurrentTWAP(t *testing.T) {
	_, addrs := sample.PrivKeyAddressPairs(5)
	app, ctx := simapp.NewTestNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper

	pair := common.MustNewAssetPair("ubtc:unusd")
	keeper.OraclesStore().AddOracles(ctx, pair, addrs)
	keeper.SetParams(ctx, types.Params{
		Pairs:              common.AssetPairs{pair},
		TwapLookbackWindow: 15 * time.Minute,
	})

	require.NoError(t, keeper.PostRawPrice(ctx, addrs[0], pair.String(), sdk.MustNewDecFromStr("0.33"), time.Now().Add(time.Hour)))
	require.NoError(t, keeper.PostRawPrice(ctx, addrs[1], pair.String(), sdk.MustNewDecFromStr("0.34"), time.Now().Add(time.Hour)))
	require.NoError(t, keeper.PostRawPrice(ctx, addrs[2], pair.String(), sdk.MustNewDecFromStr("0.35"), time.Now().Add(time.Hour)))

	// Update block time such that first 3 prices valid
	ctx = ctx.WithBlockTime(time.Now().Add(time.Minute * 45)).WithBlockHeight(1)

	// Set current price
	require.NoError(t, keeper.GatherRawPrices(ctx, pair.Token0, pair.Token1))

	// Check TWAP Price
	twap, err := keeper.GetCurrentTWAP(ctx, pair.Token0, pair.Token1)
	require.NoError(t, err)
	assert.Equal(t, sdk.MustNewDecFromStr("0.34"), twap)

	// fast forward block height as twap snapshots are indexed by blockHeight
	ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(10 * time.Second))

	require.NoError(t, keeper.PostRawPrice(ctx, addrs[4], pair.String(), sdk.MustNewDecFromStr("0.36"), time.Now().Add(time.Hour)))
	require.NoError(t, keeper.GatherRawPrices(ctx, pair.Token0, pair.Token1))

	// Check TWAP Price
	ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(10 * time.Second))

	twap, err = keeper.GetCurrentTWAP(ctx, pair.Token0, pair.Token1)
	require.NoError(t, err)
	assert.Equal(t, sdk.MustNewDecFromStr("0.3425"), twap)
}

func TestKeeper_ExpiredGatherRawPrices(t *testing.T) {
	_, oracles := sample.PrivKeyAddressPairs(5)
	app, ctx := simapp.NewTestNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper

	token0, token1 := "usd", "tst"
	pair := common.AssetPair{Token0: token0, Token1: token1}
	params := types.Params{
		Pairs: common.AssetPairs{pair},
	}
	keeper.SetParams(ctx, params)
	keeper.OraclesStore().AddOracles(ctx, pair, oracles)

	require.NoError(t, keeper.PostRawPrice(
		ctx, oracles[0], pair.String(),
		sdk.MustNewDecFromStr("0.33"),
		time.Now().Add(time.Hour*1)))

	require.NoError(t, keeper.PostRawPrice(
		ctx, oracles[1], pair.String(),
		sdk.MustNewDecFromStr("0.35"),
		time.Now().Add(time.Hour*1)))

	require.NoError(t, keeper.PostRawPrice(
		ctx, oracles[2], pair.String(),
		sdk.MustNewDecFromStr("0.34"),
		time.Now().Add(time.Hour*1)))

	// Update block time such that all prices expire
	ctx = ctx.WithBlockTime(time.Now().UTC().Add(time.Hour * 2))

	require.ErrorContains(t, keeper.GatherRawPrices(ctx, token0, token1), "input prices are expired")

	_, err := keeper.GetCurrentPrice(ctx, token0, token1)
	require.ErrorIs(t, types.ErrNoValidPrice, err, "current prices should be invalid")
}

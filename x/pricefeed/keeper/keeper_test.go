package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestKeeper_SetGetPair(t *testing.T) {
	app, ctx := testapp.NewNibiruAppAndContext(true)

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

func TestKeeper_GetSetPrice(t *testing.T) {
	app, ctx := testapp.NewNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper

	_, addrs := sample.PrivKeyAddressPairs(2)
	pair := common.MustNewAssetPair("tst:usd")
	params := types.Params{Pairs: common.AssetPairs{pair}}
	keeper.SetParams(ctx, params)
	keeper.OraclesStore().AddOracles(ctx, pair, addrs)

	priceInfos := []struct {
		oracle  sdk.AccAddress
		pairStr string
		price   sdk.Dec
		total   int
	}{
		{addrs[0], pair.String(), sdk.MustNewDecFromStr("0.33"), 1},
		{addrs[1], pair.String(), sdk.MustNewDecFromStr("0.35"), 2},
		{addrs[0], pair.String(), sdk.MustNewDecFromStr("0.37"), 2},
	}

	for _, priceInfo := range priceInfos {
		// Set price by oracle 1
		pp, err := keeper.SetPrice(
			ctx,
			priceInfo.oracle,
			priceInfo.pairStr,
			priceInfo.price,
			time.Now().UTC().Add(1*time.Hour),
		)

		require.NoError(t, err)

		// Get raw prices
		rawPrices := keeper.GetRawPrices(ctx, pair.String())

		require.Equal(t, priceInfo.total, len(rawPrices))
		require.Contains(t, rawPrices, pp)

		// Find the oracle and require price to be same
		for _, postedPrice := range rawPrices {
			oracleThatPosted, err := sdk.AccAddressFromBech32(postedPrice.Oracle)
			assert.NoError(t, err)
			if priceInfo.oracle.Equals(oracleThatPosted) {
				require.Equal(t, priceInfo.price, postedPrice.Price)
			}
		}
	}
}

/*
Test case where two oracles try to set prices for a market and only one of the
oracles is valid (i.e. registered with keeper.SetParams).
*/
func TestKeeper_SetPriceWrongOracle(t *testing.T) {
	app, ctx := testapp.NewNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper
	pair := common.MustNewAssetPair("tst:usd")

	price := sdk.MustNewDecFromStr("0.1")

	// Register addrs[1] as the oracle.
	_, addrs := sample.PrivKeyAddressPairs(2)

	params := types.Params{Pairs: common.AssetPairs{pair}}
	keeper.SetParams(ctx, params)

	// Set price with valid oracle given (addrs[0])
	keeper.WhitelistOracles(ctx, []sdk.AccAddress{addrs[0]})
	expiry := ctx.BlockTime().UTC().Add(1 * time.Hour)
	_, err := keeper.SetPrice(
		ctx, addrs[0], pair.String(), price, expiry,
	)
	require.NoError(t, err)

	// Set price with invalid oracle given (addrs[1])
	_, err = keeper.SetPrice(
		ctx, addrs[1], pair.String(), price, expiry,
	)
	require.Error(t, err)
}

/*
Test case where several oracles try to set prices for a market
and "k" (int) of the oracles are valid (i.e. registered with keeper.SetParams).
*/
func TestKeeper_SetPriceWrongOracles(t *testing.T) {
	app, ctx := testapp.NewNibiruAppAndContext(true)
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
			_, err := keeper.SetPrice(
				ctx, addr, pair.String(), price, time.Now().UTC().Add(1*time.Hour),
			)
			require.NoError(t, err)
		} else {
			// Invalid oracle addresses. This should raise errors.
			_, err := keeper.SetPrice(
				ctx, addr, pair.String(), price, time.Now().UTC().Add(1*time.Hour),
			)
			require.Error(t, err)
		}
	}
}

// TestKeeper_GetSetCurrentPrice Test Setting the median price of an Asset
func TestKeeper_GetSetCurrentPrice(t *testing.T) {
	_, addrs := sample.PrivKeyAddressPairs(5)
	app, ctx := testapp.NewNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper

	token0, token1 := "tst", "usd"
	pair := common.AssetPair{Token0: token0, Token1: token1}
	params := types.Params{
		Pairs: common.AssetPairs{pair},
	}
	keeper.OraclesStore().AddOracles(ctx, pair, addrs)
	keeper.SetParams(ctx, params)

	_, err := keeper.SetPrice(
		ctx, addrs[0], pair.String(),
		sdk.MustNewDecFromStr("0.33"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	_, err = keeper.SetPrice(
		ctx, addrs[1], pair.String(),
		sdk.MustNewDecFromStr("0.35"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	_, err = keeper.SetPrice(
		ctx, addrs[2], pair.String(),
		sdk.MustNewDecFromStr("0.34"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	t.Log("Add an expired one which should fail")
	_, err = keeper.SetPrice(
		ctx, addrs[3], pair.String(),
		sdk.MustNewDecFromStr("0.9"),
		ctx.BlockTime().Add(-time.Hour*1))
	require.Error(t, err)

	t.Log("Add a non-expired price, but will not be counted when BlockTime is changed")
	_, err = keeper.SetPrice(
		ctx, addrs[3], pair.String(),
		sdk.MustNewDecFromStr("0.9"),
		time.Now().Add(time.Minute*30))
	require.NoError(t, err)

	// Update block time such that first 3 prices valid but last one is expired
	ctx = ctx.WithBlockTime(time.Now().Add(time.Minute * 45))

	// Set current price
	err = keeper.SetCurrentPrices(ctx, token0, token1)
	require.NoError(t, err)

	// Get current price
	price, err := keeper.GetCurrentPrice(ctx, token0, token1)
	require.NoError(t, err)

	expCurPrice := sdk.MustNewDecFromStr("0.34")
	require.Truef(
		t,
		price.Price.Equal(expCurPrice),
		"expected current price to equal %s, actual %s",
		expCurPrice, price.Price,
	)

	// Even number of oracles
	_, err = keeper.SetPrice(
		ctx, addrs[4], pair.String(),
		sdk.MustNewDecFromStr("0.36"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	err = keeper.SetCurrentPrices(ctx, token0, token1)
	require.NoError(t, err)

	price, err = keeper.GetCurrentPrice(ctx, "tst", "usd")
	require.Nil(t, err)

	exp := sdk.MustNewDecFromStr("0.345")
	require.Truef(t, price.Price.Equal(exp),
		"current price %s should be %s",
		price.Price.String(),
		exp.String(),
	)

	prices := keeper.GetCurrentPrices(ctx)
	require.Equal(t, 1, len(prices))
	require.Equal(t, price, prices[0])
}

func TestKeeper_ExpiredSetCurrentPrices(t *testing.T) {
	_, oracles := sample.PrivKeyAddressPairs(5)
	app, ctx := testapp.NewNibiruAppAndContext(true)
	keeper := app.PricefeedKeeper

	token0, token1 := "usd", "tst"
	pair := common.AssetPair{Token0: token0, Token1: token1}
	params := types.Params{
		Pairs: common.AssetPairs{pair},
	}
	keeper.SetParams(ctx, params)
	keeper.OraclesStore().AddOracles(ctx, pair, oracles)

	_, err := keeper.SetPrice(
		ctx, oracles[0], pair.String(),
		sdk.MustNewDecFromStr("0.33"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	_, err = keeper.SetPrice(
		ctx, oracles[1], pair.String(),
		sdk.MustNewDecFromStr("0.35"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	_, err = keeper.SetPrice(
		ctx, oracles[2], pair.String(),
		sdk.MustNewDecFromStr("0.34"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	// Update block time such that all prices expire
	ctx = ctx.WithBlockTime(time.Now().UTC().Add(time.Hour * 2))

	err = keeper.SetCurrentPrices(ctx, token0, token1)
	require.ErrorContains(t, err, "input prices are expired")

	_, err = keeper.GetCurrentPrice(ctx, token0, token1)
	require.ErrorIs(t, types.ErrNoValidPrice, err, "current prices should be invalid")
}

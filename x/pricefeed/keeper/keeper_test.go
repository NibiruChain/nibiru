package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestKeeper_SetGetPair(t *testing.T) {
	app, ctx := testutil.NewNibiruApp(true)

	token0, token1 := "usd", "tst"
	tstusdPair := types.Pair{
		Token1: token1, Token0: token0, Oracles: []sdk.AccAddress{}, Active: true}
	token0, token1 = "abc", "xyz"
	abcxyzPair := types.Pair{
		Token1: token1, Token0: token0, Oracles: []sdk.AccAddress{}, Active: true}

	mp := types.Params{
		Pairs: types.Pairs{tstusdPair},
	}
	keeper := app.PriceKeeper
	keeper.SetParams(ctx, mp)

	markets := keeper.GetPairs(ctx)
	require.Equal(t, len(markets), 1)
	require.Equal(t, markets[0].PairID(), "tst:usd")

	_, found := keeper.GetPair(ctx, "tst:usd")
	require.True(t, found, "market should be found")

	_, found = keeper.GetPair(ctx, "invalid:market")
	require.False(t, found, "invalid:market should not be found")

	mp = types.Params{
		Pairs: []types.Pair{
			tstusdPair,
			abcxyzPair,
		},
	}

	keeper.SetParams(ctx, mp)
	markets = keeper.GetPairs(ctx)
	require.Equal(t, len(markets), 2)
	require.Equal(t, markets[0].PairID(), "tst:usd")
	_, found = keeper.GetPair(ctx, "abc:xyz")
	require.True(t, found, "market should be found")

	_, found = keeper.GetPair(ctx, "nan")
	require.False(t, found)
}

func TestKeeper_GetSetPrice(t *testing.T) {
	app, ctx := testutil.NewNibiruApp(true)
	keeper := app.PriceKeeper

	_, addrs := sample.PrivKeyAddressPairs(2)
	mp := types.Params{
		Pairs: []types.Pair{
			{Token1: "tst", Token0: "usd", Oracles: addrs, Active: true},
		},
	}
	keeper.SetParams(ctx, mp)

	token0, token1 := "tst", "usd"
	pairID := "tst:usd"
	prices := []struct {
		oracle sdk.AccAddress
		token0 string
		token1 string
		price  sdk.Dec
		total  int
	}{
		{addrs[0], token0, token1, sdk.MustNewDecFromStr("0.33"), 1},
		{addrs[1], token0, token1, sdk.MustNewDecFromStr("0.35"), 2},
		{addrs[0], token0, token1, sdk.MustNewDecFromStr("0.37"), 2},
	}

	for _, p := range prices {
		// Set price by oracle 1

		pp, err := keeper.SetPrice(
			ctx,
			p.oracle,
			p.token0,
			p.token1,
			p.price,
			time.Now().UTC().Add(1*time.Hour),
		)

		require.NoError(t, err)

		// Get raw prices
		rawPrices := keeper.GetRawPrices(ctx, pairID)

		require.Equal(t, p.total, len(rawPrices))
		require.Contains(t, rawPrices, pp)

		// Find the oracle and require price to be same
		for _, rp := range rawPrices {
			if p.oracle.Equals(rp.OracleAddress) {
				require.Equal(t, p.price, rp.Price)
			}
		}
	}
}

/*
Test case where two oracles try to set prices for a market and only one of the
oracles is valid (i.e. registered with keeper.SetParams).
*/
func TestKeeper_SetPriceWrongOracle(t *testing.T) {
	app, ctx := testutil.NewNibiruApp(true)
	keeper := app.PriceKeeper
	token0, token1 := "tst", "usd"
	price := sdk.MustNewDecFromStr("0.1")

	// Register addrs[1] as the oracle.
	_, addrs := sample.PrivKeyAddressPairs(2)
	mp := types.Params{
		Pairs: []types.Pair{
			{Token1: token1, Token0: token0,
				Oracles: addrs[:1], Active: true},
		}}
	keeper.SetParams(ctx, mp)

	// Set price with valid oracle given (addrs[1])
	_, err := keeper.SetPrice(
		ctx, addrs[0], token0, token1, price, time.Now().UTC().Add(1*time.Hour),
	)
	require.NoError(t, err)

	// Set price with invalid oracle given (addrs[1])
	_, err = keeper.SetPrice(
		ctx, addrs[1], token0, token1, price, time.Now().UTC().Add(1*time.Hour),
	)
	require.Error(t, err)
}

/*
Test case where several oracles try to set prices for a market
and "k" (int) of the oracles are valid (i.e. registered with keeper.SetParams).
*/
func TestKeeper_SetPriceWrongOracles(t *testing.T) {
	app, ctx := testutil.NewNibiruApp(true)
	keeper := app.PriceKeeper

	token0, token1 := "tst", "usd"
	price := sdk.MustNewDecFromStr("0.1")

	_, addrs := sample.PrivKeyAddressPairs(10)
	mp := types.Params{
		Pairs: []types.Pair{
			{Token1: "tst", Token0: "usd",
				Oracles: addrs[:5], Active: true},
		},
	}
	keeper.SetParams(ctx, mp)

	for i, addr := range addrs {
		if i < 5 {
			// Valid oracle addresses. This shouldn't raise an error.
			_, err := keeper.SetPrice(
				ctx, addr, token0, token1, price, time.Now().UTC().Add(1*time.Hour),
			)
			require.NoError(t, err)
		} else {
			// Invalid oracle addresses. This should raise errors.
			_, err := keeper.SetPrice(
				ctx, addr, token0, token1, price, time.Now().UTC().Add(1*time.Hour),
			)
			require.Error(t, err)
		}
	}
}

// TestKeeper_GetSetCurrentPrice Test Setting the median price of an Asset
func TestKeeper_GetSetCurrentPrice(t *testing.T) {
	_, addrs := sample.PrivKeyAddressPairs(5)
	app, ctx := testutil.NewNibiruApp(true)
	keeper := app.PriceKeeper

	token0, token1 := "tst", "usd"
	mp := types.Params{
		Pairs: []types.Pair{
			{Token0: token0, Token1: token1,
				Oracles: addrs, Active: true},
		},
	}
	keeper.SetParams(ctx, mp)

	_, err := keeper.SetPrice(
		ctx, addrs[0], token0, token1,
		sdk.MustNewDecFromStr("0.33"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	_, err = keeper.SetPrice(
		ctx, addrs[1], token0, token1,
		sdk.MustNewDecFromStr("0.35"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	_, err = keeper.SetPrice(
		ctx, addrs[2], token0, token1,
		sdk.MustNewDecFromStr("0.34"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	// Add an expired one which should fail
	_, err = keeper.SetPrice(
		ctx, addrs[3], token0, token1,
		sdk.MustNewDecFromStr("0.9"),
		ctx.BlockTime().Add(-time.Hour*1))
	require.Error(t, err)

	// Add a non-expired price, but will not be counted when BlockTime is changed
	_, err = keeper.SetPrice(
		ctx, addrs[3], token0, token1,
		sdk.MustNewDecFromStr("0.9"),
		time.Now().Add(time.Minute*30))
	require.NoError(t, err)

	// Update block time such that first 3 prices valid but last one is expired
	ctx = ctx.WithBlockTime(time.Now().Add(time.Minute * 45))

	// Set current price
	err = keeper.SetCurrentPrices(ctx, "tst", "usd")
	require.NoError(t, err)

	// Get current price
	price, err := keeper.GetCurrentPrice(ctx, "tst", "usd")
	require.Nil(t, err)

	expCurPrice := sdk.MustNewDecFromStr("0.34")
	require.Truef(
		t,
		price.Price.Equal(expCurPrice),
		"expected current price to equal %v, actual %v",
		expCurPrice, price.Price,
	)

	// Even number of oracles
	_, err = keeper.SetPrice(
		ctx, addrs[4], token0, token1,
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
	_, addrs := sample.PrivKeyAddressPairs(5)
	app, ctx := testutil.NewNibiruApp(true)
	keeper := app.PriceKeeper

	token0, token1 := "usd", "tst"
	mp := types.Params{
		Pairs: []types.Pair{
			{Token1: token1, Token0: token0, Oracles: addrs, Active: true},
		},
	}
	keeper.SetParams(ctx, mp)

	_, err := keeper.SetPrice(
		ctx, addrs[0], token0, token1,
		sdk.MustNewDecFromStr("0.33"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	_, err = keeper.SetPrice(
		ctx, addrs[1], token0, token1,
		sdk.MustNewDecFromStr("0.35"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	_, err = keeper.SetPrice(
		ctx, addrs[2], token0, token1,
		sdk.MustNewDecFromStr("0.34"),
		time.Now().Add(time.Hour*1))
	require.NoError(t, err)

	// Update block time such that all prices expire
	ctx = ctx.WithBlockTime(time.Now().UTC().Add(time.Hour * 2))

	err = keeper.SetCurrentPrices(ctx, token0, token1)
	require.ErrorIs(t, types.ErrNoValidPrice, err, "there should be no valid prices to be set")

	_, err = keeper.GetCurrentPrice(ctx, token0, token1)
	require.ErrorIs(t, types.ErrNoValidPrice, err, "current prices should be invalid")
}

package pricefeed_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed"
	pricefeedkeeper "github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	ptypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestTWAPriceUpdates(t *testing.T) {
	var nibiruApp *simapp.NibiruTestApp
	var ctx sdk.Context

	oracle := sample.AccAddress()
	pair := common.AssetPair{
		Token0: common.DenomUSDC,
		Token1: common.DenomNUSD,
	}

	runBlock := func(duration time.Duration) {
		ctx = ctx.
			WithBlockHeight(ctx.BlockHeight() + 1).
			WithBlockTime(ctx.BlockTime().Add(duration))
		pricefeed.BeginBlocker(ctx, nibiruApp.PricefeedKeeper)
	}
	setPrice := func(price string) {
		require.NoError(t, nibiruApp.PricefeedKeeper.PostRawPrice(
			ctx, oracle, pair.String(),
			sdk.MustNewDecFromStr(price), ctx.BlockTime().Add(time.Hour*5000*4)))
	}

	nibiruApp, ctx = simapp.NewTestNibiruAppAndContext(true)

	ctx = ctx.WithBlockTime(time.Date(2015, 10, 21, 0, 0, 0, 0, time.UTC))

	oracles := []sdk.AccAddress{oracle}
	pairs := common.AssetPairs{pair}
	params := ptypes.NewParams(pairs, 15_001*time.Hour)
	nibiruApp.PricefeedKeeper.SetParams(ctx, params) // makes pairs active
	nibiruApp.PricefeedKeeper.WhitelistOraclesForPairs(ctx, oracles, pairs)

	// Sim set price set the price for one hour
	setPrice("0.9")

	err := nibiruApp.StablecoinKeeper.SetCollRatio(ctx, sdk.MustNewDecFromStr("0.8"))
	require.NoError(t, err)

	t.Log("Pass 5000 hours. Previous price is live and we post a new one")
	runBlock(time.Hour * 5000)
	setPrice("0.8")
	runBlock(time.Hour * 5000)

	/*
		New price should be.

		deltaT1: 10minutes / 600_000
		deltaT2: 5000hours / 18_000_000_000

		(0.8 * 600_000 + 0.9 * 18_000_000_000) / 18_000_600_000 = 0.899996666777774074
	*/

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Minute))
	nibiruApp.PricefeedKeeper.SetParams(ctx, params)
	price, err := nibiruApp.PricefeedKeeper.GetCurrentTWAP(
		ctx, pair.Token0, pair.Token1)
	require.NoError(t, err)
	priceFloat, err := price.Float64()
	require.NoError(t, err)
	require.InDelta(t, 0.899996666777774074, priceFloat, 0.03)

	t.Log("5000 hours passed, both previous prices are live and we post a new one")
	setPrice("0.82")
	runBlock(time.Hour * 5000)

	/*
		New price should be.

		deltaT1: 10min / 600_000
		deltaT2: 5000h + 10min / 18_000_600_000
		deltaT3: 5000h + 10min / 18_000_600_000

		(0.82 * 600_000 + 0.8 * 18_000_600_000 + 0.9 * 18_000_000_000) / 36_001_200_000 = 0.849998666711109629
	*/
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Minute))
	price, err = nibiruApp.PricefeedKeeper.GetCurrentTWAP(
		ctx, pair.Token0, pair.Token1)
	require.NoError(t, err)
	priceFloat, err = price.Float64()
	require.NoError(t, err)
	require.InDelta(t, 0.849998666711109629, priceFloat, 0.02)

	// 5000 hours passed, first price is now expired
	/*
		New price should be.

		deltaT1: 10min / 600_000
		deltaT2: 5000h + 10min / 18_000_600_000
		deltaT3: 5000h + 10min/ 18_000_600_000
		deltaT4: 5000h + 10min/ 18_000_600_000

		(0.83 * 600_000 + 0.82 * 18_000_600_000 + 0.8 * 18_000_600_000 + 0.9 * 18_000_000_000) / 54_001_800_000 = 0.839999222248147283
	*/
	setPrice("0.83")
	runBlock(time.Hour * 5000)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Minute))
	price, err = nibiruApp.PricefeedKeeper.GetCurrentTWAP(
		ctx, pair.Token0, pair.Token1)

	require.NoError(t, err)
	priceFloat, err = price.Float64()
	require.NoError(t, err)
	require.InDelta(t, 0.839999222248147283, priceFloat, 0.01)

	// ensure the lookbackInterval is properly adhered
	/*
		New price should be.

		deltaT1: 10min / 600_000
		deltaT2: 5000h + 10min / 18_000_600_000
		deltaT4: 2000h - 20min / 7_198_800_000

		(0.83 * 600_000 + 0.82 * 18_000_600_000 + 0.8 * 7_198_800_000) / 25_200_000_000 = 0.839999222248147283
	*/
	setLookbackWindow(ctx, nibiruApp.PricefeedKeeper, 7_000*time.Hour)
	price, err = nibiruApp.PricefeedKeeper.GetCurrentTWAP(
		ctx, pair.Token0, pair.Token1)

	require.NoError(t, err)
	priceFloat, err = price.Float64()
	require.NoError(t, err)
	require.InDelta(t, 0.814286904761904761, priceFloat, 0.01)
}

func setLookbackWindow(ctx sdk.Context, pfk pricefeedkeeper.Keeper, d time.Duration) {
	params := pfk.GetParams(ctx)
	params.TwapLookbackWindow = d
	pfk.SetParams(ctx, params)
}

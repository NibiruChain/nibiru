package pricefeed_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed"
	ptypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestTWAPriceUpdates(t *testing.T) {
	var nibiruApp *app.NibiruApp
	var ctx sdk.Context

	oracle := sample.AccAddress()
	pair := common.AssetPair{
		Token0: common.DenomColl,
		Token1: common.DenomStable,
	}

	runBlock := func(duration time.Duration) {
		ctx = ctx.
			WithBlockHeight(ctx.BlockHeight() + 1).
			WithBlockTime(ctx.BlockTime().Add(duration))
		pricefeed.BeginBlocker(ctx, nibiruApp.PricefeedKeeper)
	}
	setPrice := func(price string) {
		_, err := nibiruApp.PricefeedKeeper.PostRawPrice(
			ctx, oracle, pair.String(),
			sdk.MustNewDecFromStr(price), ctx.BlockTime().Add(time.Hour*5000*4))
		require.NoError(t, err)
	}

	nibiruApp, ctx = testapp.NewNibiruAppAndContext(true)

	ctx = ctx.WithBlockTime(time.Date(2015, 10, 21, 0, 0, 0, 0, time.UTC))

	oracles := []sdk.AccAddress{oracle}
	pairs := common.AssetPairs{pair}
	params := ptypes.NewParams(pairs)
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

		T0: 1463385600
		T1: 1481385600

		(0.9 * 1463385600 + (0.9 + 0.8) / 2 * 1481385600) / (1463385600 + 1481385600) = 0.8749844622444971
	*/

	price, err := nibiruApp.PricefeedKeeper.GetCurrentTWAP(
		ctx, pair.Token0, pair.Token1)
	require.NoError(t, err)
	priceFloat, err := price.Price.Float64()
	require.NoError(t, err)
	require.InDelta(t, 0.8748471867695528, priceFloat, 0.03)

	t.Log("5000 hours passed, both previous prices are live and we post a new one")
	setPrice("0.82")
	runBlock(time.Hour * 5000)

	/*
		New price should be.

		T0: 1463385600
		T1: 1481385600
		T1: 1499385600

		(0.9 * 1463385600 + (0.9 + 0.8) / 2 * 1481385600 + 0.82 * 1499385600) / (1463385600 + 1481385600 + 1499385600) = 0.8563426456960295
	*/
	price, err = nibiruApp.PricefeedKeeper.GetCurrentTWAP(
		ctx, pair.Token0, pair.Token1)
	require.NoError(t, err)
	priceFloat, err = price.Price.Float64()
	require.NoError(t, err)
	require.InDelta(t, 0.8563426456960295, priceFloat, 0.02)

	// 5000 hours passed, first price is now expired
	/*
		New price should be.

		T0: 1463385600
		T1: 1481385600
		T1: 1499385600
		T4: 1517385600

		(0.9 * 1463385600 + (0.9 + 0.8) / 2 * 1481385600 + 0.82 * 1499385600 + .82 * 1517385600) / (1463385600 + 1481385600 + 1499385600 + 1517385600) = 0.8470923873660615
	*/
	setPrice("0.83")
	runBlock(time.Hour * 5000)
	price, err = nibiruApp.PricefeedKeeper.GetCurrentTWAP(
		ctx, pair.Token0, pair.Token1)

	require.NoError(t, err)
	priceFloat, err = price.Price.Float64()
	require.NoError(t, err)
	require.InDelta(t, 0.8470923873660615, priceFloat, 0.01)
}

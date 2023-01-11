package keeper_test

import (
	"testing"

	dexkeeper "github.com/NibiruChain/nibiru/x/dex/keeper"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSetDenomLiquidity(t *testing.T) {
	app, ctx := testapp.NewTestNibiruAppAndContext(true)

	// Write to store
	coin := sdk.NewCoin("nibi", sdk.NewInt(1_000))
	assert.NoError(t, app.DexKeeper.SetDenomLiquidity(ctx, coin.Denom, coin.Amount))

	// Read from store
	amount, err := app.DexKeeper.GetDenomLiquidity(ctx, "nibi")
	assert.NoError(t, err)
	require.EqualValues(t, sdk.NewInt(1000), amount)
}

func TestGetTotalLiquidity(t *testing.T) {
	app, ctx := testapp.NewTestNibiruAppAndContext(true)

	// Write to store
	coinMap := map[string]sdk.Int{
		"atom": sdk.NewInt(123),
		"nibi": sdk.NewInt(456),
		"foo":  sdk.NewInt(789),
	}
	for denom, amount := range coinMap {
		coin := sdk.NewCoin(denom, amount)
		assert.NoError(t, app.DexKeeper.SetDenomLiquidity(ctx, coin.Denom, coin.Amount))
	}

	// Read from store
	coins := app.DexKeeper.GetTotalLiquidity(ctx)

	require.EqualValues(t, sdk.NewCoins(
		sdk.NewCoin("atom", coinMap["atom"]),
		sdk.NewCoin("nibi", coinMap["nibi"]),
		sdk.NewCoin("foo", coinMap["foo"]),
	), coins)
}

// assertLiqValues checks if the total liquidity for each denom (key) of the
// expected map matches what's given by the DexKeeper
func assertLiqValues(
	t *testing.T,
	ctx sdk.Context,
	dexKeeper dexkeeper.Keeper,
	expected map[string]sdk.Int,
) {
	for denom, expectedLiq := range expected {
		liq, err := dexKeeper.GetDenomLiquidity(ctx, denom)
		assert.NoError(t, err)
		assert.EqualValues(t, liq, expectedLiq)
	}
}

func TestSetTotalLiquidity(t *testing.T) {
	app, ctx := testapp.NewTestNibiruAppAndContext(true)

	// Write to store
	assert.NoError(t, app.DexKeeper.SetTotalLiquidity(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(123)),
		sdk.NewCoin("nibi", sdk.NewInt(456)),
		sdk.NewCoin("foo", sdk.NewInt(789)),
	)))

	// Read from store
	expectedLiqValues := map[string]sdk.Int{
		"atom": sdk.NewInt(123),
		"nibi": sdk.NewInt(456),
		"foo":  sdk.NewInt(789),
	}
	assertLiqValues(t, ctx, app.DexKeeper, expectedLiqValues)
}

func TestRecordTotalLiquidityIncrease(t *testing.T) {
	app, ctx := testapp.NewTestNibiruAppAndContext(true)

	// Write to store
	assert.NoError(t, app.DexKeeper.SetTotalLiquidity(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(100)),
		sdk.NewCoin("nibi", sdk.NewInt(200)),
	)))
	err := app.DexKeeper.RecordTotalLiquidityIncrease(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(50)),
		sdk.NewCoin("nibi", sdk.NewInt(75)),
	))
	assert.NoError(t, err)

	expectedLiqValues := map[string]sdk.Int{
		"atom": sdk.NewInt(150),
		"nibi": sdk.NewInt(275),
	}
	assertLiqValues(t, ctx, app.DexKeeper, expectedLiqValues)
}

func TestRecordTotalLiquidityDecrease(t *testing.T) {
	app, ctx := testapp.NewTestNibiruAppAndContext(true)

	// Write to store
	assert.NoError(t, app.DexKeeper.SetTotalLiquidity(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(100)),
		sdk.NewCoin("nibi", sdk.NewInt(200)),
	)))
	err := app.DexKeeper.RecordTotalLiquidityDecrease(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(50)),
		sdk.NewCoin("nibi", sdk.NewInt(75)),
	))
	assert.NoError(t, err)

	expectedLiqValues := map[string]sdk.Int{
		"atom": sdk.NewInt(50),
		"nibi": sdk.NewInt(125),
	}
	assertLiqValues(t, ctx, app.DexKeeper, expectedLiqValues)
}

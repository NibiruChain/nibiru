package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/spot/keeper"
)

func TestGetSetDenomLiquidity(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	// Write to store
	coin := sdk.NewCoin("nibi", sdk.NewInt(1_000))
	assert.NoError(t, app.SpotKeeper.SetDenomLiquidity(ctx, coin.Denom, coin.Amount))

	// Read from store
	amount, err := app.SpotKeeper.GetDenomLiquidity(ctx, "nibi")
	assert.NoError(t, err)
	require.EqualValues(t, sdk.NewInt(1000), amount)
}

func TestGetTotalLiquidity(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	// Write to store
	coinMap := map[string]sdkmath.Int{
		"atom": sdk.NewInt(123),
		"nibi": sdk.NewInt(456),
		"foo":  sdk.NewInt(789),
	}
	for denom, amount := range coinMap {
		coin := sdk.NewCoin(denom, amount)
		assert.NoError(t, app.SpotKeeper.SetDenomLiquidity(ctx, coin.Denom, coin.Amount))
	}

	// Read from store
	coins := app.SpotKeeper.GetTotalLiquidity(ctx)

	require.EqualValues(t, sdk.NewCoins(
		sdk.NewCoin("atom", coinMap["atom"]),
		sdk.NewCoin("nibi", coinMap["nibi"]),
		sdk.NewCoin("foo", coinMap["foo"]),
	), coins)
}

// assertLiqValues checks if the total liquidity for each denom (key) of the
// expected map matches what's given by the SpotKeeper
func assertLiqValues(
	t *testing.T,
	ctx sdk.Context,
	spotKeeper keeper.Keeper,
	expected map[string]sdkmath.Int,
) {
	for denom, expectedLiq := range expected {
		liq, err := spotKeeper.GetDenomLiquidity(ctx, denom)
		assert.NoError(t, err)
		assert.EqualValues(t, liq, expectedLiq)
	}
}

func TestSetTotalLiquidity(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	// Write to store
	assert.NoError(t, app.SpotKeeper.SetTotalLiquidity(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(123)),
		sdk.NewCoin("nibi", sdk.NewInt(456)),
		sdk.NewCoin("foo", sdk.NewInt(789)),
	)))

	// Read from store
	expectedLiqValues := map[string]sdkmath.Int{
		"atom": sdk.NewInt(123),
		"nibi": sdk.NewInt(456),
		"foo":  sdk.NewInt(789),
	}
	assertLiqValues(t, ctx, app.SpotKeeper, expectedLiqValues)
}

func TestRecordTotalLiquidityIncrease(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	// Write to store
	assert.NoError(t, app.SpotKeeper.SetTotalLiquidity(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(100)),
		sdk.NewCoin("nibi", sdk.NewInt(200)),
	)))
	err := app.SpotKeeper.RecordTotalLiquidityIncrease(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(50)),
		sdk.NewCoin("nibi", sdk.NewInt(75)),
	))
	assert.NoError(t, err)

	expectedLiqValues := map[string]sdkmath.Int{
		"atom": sdk.NewInt(150),
		"nibi": sdk.NewInt(275),
	}
	assertLiqValues(t, ctx, app.SpotKeeper, expectedLiqValues)
}

func TestRecordTotalLiquidityDecrease(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	// Write to store
	assert.NoError(t, app.SpotKeeper.SetTotalLiquidity(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(100)),
		sdk.NewCoin("nibi", sdk.NewInt(200)),
	)))
	err := app.SpotKeeper.RecordTotalLiquidityDecrease(ctx, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(50)),
		sdk.NewCoin("nibi", sdk.NewInt(75)),
	))
	assert.NoError(t, err)

	expectedLiqValues := map[string]sdkmath.Int{
		"atom": sdk.NewInt(50),
		"nibi": sdk.NewInt(125),
	}
	assertLiqValues(t, ctx, app.SpotKeeper, expectedLiqValues)
}

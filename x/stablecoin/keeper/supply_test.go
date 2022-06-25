package keeper_test

import (
	"testing"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	dextypes "github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/stablecoin/mock"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestKeeper_GetStableMarketCap(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruAppAndContext(false)
	k := nibiruApp.StablecoinKeeper

	// We set some supply
	err := k.BankKeeper.MintCoins(ctx, types.ModuleName, sdktypes.NewCoins(
		sdktypes.NewInt64Coin(common.DenomStable, 1_000_000),
	))
	require.NoError(t, err)

	// We set some supply
	marketCap := k.GetStableMarketCap(ctx)

	require.Equal(t, sdktypes.NewInt(1_000_000), marketCap)
}

func TestKeeper_GetGovMarketCap(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruAppAndContext(false)
	keeper := nibiruApp.StablecoinKeeper

	poolAccountAddr := sample.AccAddress()
	poolParams := dextypes.PoolParams{
		SwapFee: sdktypes.NewDecWithPrec(3, 2),
		ExitFee: sdktypes.NewDecWithPrec(3, 2),
	}
	poolAssets := []dextypes.PoolAsset{
		{
			Token:  sdktypes.NewInt64Coin(common.DenomGov, 2_000_000),
			Weight: sdktypes.NewInt(100),
		},
		{
			Token:  sdktypes.NewInt64Coin(common.DenomStable, 1_000_000),
			Weight: sdktypes.NewInt(100),
		},
	}

	pool, err := dextypes.NewPool(1, poolAccountAddr, poolParams, poolAssets)
	require.NoError(t, err)
	keeper.DexKeeper = mock.NewKeeper(pool)

	// We set some supply
	err = keeper.BankKeeper.MintCoins(ctx, types.ModuleName, sdktypes.NewCoins(
		sdktypes.NewInt64Coin(common.DenomGov, 1_000_000),
	))
	require.NoError(t, err)

	marketCap, err := keeper.GetGovMarketCap(ctx)
	require.NoError(t, err)

	require.Equal(t, sdktypes.NewInt(2_000_000), marketCap) // 1 * 10^6 * 2 (price of gov token)
}

func TestKeeper_GetLiquidityRatio_AndBands(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruAppAndContext(false)
	keeper := nibiruApp.StablecoinKeeper

	poolAccountAddr := sample.AccAddress()
	poolParams := dextypes.PoolParams{
		SwapFee: sdktypes.NewDecWithPrec(3, 2),
		ExitFee: sdktypes.NewDecWithPrec(3, 2),
	}
	poolAssets := []dextypes.PoolAsset{
		{
			Token:  sdktypes.NewInt64Coin(common.DenomGov, 2_000_000),
			Weight: sdktypes.NewInt(100),
		},
		{
			Token:  sdktypes.NewInt64Coin(common.DenomStable, 1_000_000),
			Weight: sdktypes.NewInt(100),
		},
	}

	pool, err := dextypes.NewPool(1, poolAccountAddr, poolParams, poolAssets)
	require.NoError(t, err)
	keeper.DexKeeper = mock.NewKeeper(pool)

	// We set some supply
	err = keeper.BankKeeper.MintCoins(ctx, types.ModuleName, sdktypes.NewCoins(
		sdktypes.NewInt64Coin(common.DenomGov, 1_000_000),
	))
	require.NoError(t, err)

	err = keeper.BankKeeper.MintCoins(ctx, types.ModuleName, sdktypes.NewCoins(
		sdktypes.NewInt64Coin(common.DenomStable, 1_000_000),
	))
	require.NoError(t, err)

	liquidityRatio, err := keeper.GetLiquidityRatio(ctx)
	require.NoError(t, err)

	require.Equal(t, sdktypes.MustNewDecFromStr("2"), liquidityRatio) // 2 * 1 * 10^6 / Stable 1 * 10^6

	lowBand, upBand, err := keeper.GetLiquidityRatioBands(ctx)
	require.NoError(t, err)

	require.Equal(t, sdktypes.MustNewDecFromStr("1.998"), lowBand)
	require.Equal(t, sdktypes.MustNewDecFromStr("2.002"), upBand)
}

package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/testutil"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	dextypes "github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/stablecoin/mock"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

func TestKeeper_GetStableMarketCap(t *testing.T) {
	nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(false)
	k := nibiruApp.StablecoinKeeper

	// We set some supply
	err := k.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(
		sdk.NewInt64Coin(denoms.NUSD, 1*common.Precision),
	))
	require.NoError(t, err)

	// We set some supply
	marketCap := k.GetStableMarketCap(ctx)

	require.Equal(t, sdk.NewInt(1*common.Precision), marketCap)
}

func TestKeeper_GetGovMarketCap(t *testing.T) {
	nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(false)
	keeper := nibiruApp.StablecoinKeeper

	poolAccountAddr := testutil.AccAddress()
	poolParams := dextypes.PoolParams{
		SwapFee:  sdk.NewDecWithPrec(3, 2),
		ExitFee:  sdk.NewDecWithPrec(3, 2),
		PoolType: dextypes.PoolType_BALANCER,
	}
	poolAssets := []dextypes.PoolAsset{
		{
			Token:  sdk.NewInt64Coin(denoms.NIBI, 2*common.Precision),
			Weight: sdk.NewInt(100),
		},
		{
			Token:  sdk.NewInt64Coin(denoms.NUSD, 1*common.Precision),
			Weight: sdk.NewInt(100),
		},
	}

	pool, err := dextypes.NewPool(1, poolAccountAddr, poolParams, poolAssets)
	require.NoError(t, err)
	keeper.DexKeeper = mock.NewKeeper(pool)

	// We set some supply
	err = keeper.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(
		sdk.NewInt64Coin(denoms.NIBI, 1*common.Precision),
	))
	require.NoError(t, err)

	marketCap, err := keeper.GetGovMarketCap(ctx)
	require.NoError(t, err)

	require.Equal(t, sdk.NewInt(2*common.Precision), marketCap) // 1 * 10^6 * 2 (price of gov token)
}

func TestKeeper_GetLiquidityRatio_AndBands(t *testing.T) {
	nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(false)
	keeper := nibiruApp.StablecoinKeeper

	poolAccountAddr := testutil.AccAddress()
	poolParams := dextypes.PoolParams{
		SwapFee:  sdk.NewDecWithPrec(3, 2),
		ExitFee:  sdk.NewDecWithPrec(3, 2),
		PoolType: dextypes.PoolType_BALANCER,
	}
	poolAssets := []dextypes.PoolAsset{
		{
			Token:  sdk.NewInt64Coin(denoms.NIBI, 2*common.Precision),
			Weight: sdk.NewInt(100),
		},
		{
			Token:  sdk.NewInt64Coin(denoms.NUSD, 1*common.Precision),
			Weight: sdk.NewInt(100),
		},
	}

	pool, err := dextypes.NewPool(1, poolAccountAddr, poolParams, poolAssets)
	require.NoError(t, err)
	keeper.DexKeeper = mock.NewKeeper(pool)

	// We set some supply
	err = keeper.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(
		sdk.NewInt64Coin(denoms.NIBI, 1*common.Precision),
	))
	require.NoError(t, err)

	err = keeper.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(
		sdk.NewInt64Coin(denoms.NUSD, 1*common.Precision),
	))
	require.NoError(t, err)

	liquidityRatio, err := keeper.GetLiquidityRatio(ctx)
	require.NoError(t, err)

	require.Equal(t, sdk.MustNewDecFromStr("2"), liquidityRatio) // 2 * 1 * 10^6 / Stable 1 * 10^6

	lowBand, upBand, err := keeper.GetLiquidityRatioBands(ctx)
	require.NoError(t, err)

	require.Equal(t, sdk.MustNewDecFromStr("1.998"), lowBand)
	require.Equal(t, sdk.MustNewDecFromStr("2.002"), upBand)
}

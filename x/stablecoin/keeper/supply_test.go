package keeper_test

import (
	"testing"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestKeeper_GetStableMarketCap(t *testing.T) {
	matrixApp, ctx := testutil.NewNibiruApp(false)
	k := matrixApp.StablecoinKeeper

	// We set some supply
	err := k.BankKeeper.MintCoins(ctx, types.ModuleName, sdktypes.NewCoins(
		sdktypes.NewInt64Coin(common.StableDenom, 1_000_000),
	))
	require.NoError(t, err)

	// We set some supply
	marketCap := k.GetStableMarketCap(ctx)

	require.Equal(t, sdktypes.NewInt(1_000_000), marketCap)
}

func TestKeeper_GetGovMarketCap(t *testing.T) {
	matrixApp, ctx := testutil.NewNibiruApp(false)
	keeper := matrixApp.StablecoinKeeper

	// We set some supply
	err := keeper.BankKeeper.MintCoins(ctx, types.ModuleName, sdktypes.NewCoins(
		sdktypes.NewInt64Coin(common.GovDenom, 1_000_000),
	))
	require.NoError(t, err)

	// We set some supply
	marketCap := keeper.GetGovMarketCap(ctx)

	require.Equal(t, sdktypes.NewInt(1_000_000), marketCap)
}

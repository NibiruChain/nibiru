package keeper_test

import (
	"testing"

	testkeeper "github.com/MatrixDAO/dex/testutil/keeper"
	"github.com/MatrixDAO/dex/x/stablecoin/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.StablecoinKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

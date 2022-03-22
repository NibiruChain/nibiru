package keeper_test

import (
	"testing"

	testkeeper "github.com/MatrixDao/dex/testutil/keeper"
	"github.com/MatrixDao/dex/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.DexKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

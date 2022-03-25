package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/testutil"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testutil.StablecoinKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	matrixApp, ctx := testutil.NewMatrixApp()
	stableKeeper := &matrixApp.StablecoinKeeper

	params := types.DefaultParams()

	stableKeeper.SetParams(ctx, params)

	require.EqualValues(t, params, stableKeeper.GetParams(ctx))
}

package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/testutil"
	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	app, ctx := testutil.NewApp()

	params := types.DefaultParams()
	app.DexKeeper.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

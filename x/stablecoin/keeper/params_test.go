package keeper_test

import (
	"fmt"
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	// "github.com/MatrixDao/matrix/x/testutil"
	// "github.com/MatrixDao/matrix/x/testutil/testkeeper"
	"github.com/MatrixDao/matrix/x/testutil/testkeeper"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	// matrixApp, ctx := testutil.NewMatrixApp()
	// stableKeeper := &matrixApp.StablecoinKeeper
	stableKeeper, ctx := testkeeper.StablecoinKeeper(t)

	params := types.DefaultParams()

	stableKeeper.SetParams(ctx, params)
	fmt.Println(stableKeeper.ParamSubspace)

	require.EqualValues(t, params, stableKeeper.GetParams(ctx))
}

package keeper_test

import (
	"testing"

	testkeeper "github.com/MatrixDao/matrix/testutil/keeper"
	"github.com/MatrixDao/matrix/x/derivatives/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.DerivativesKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

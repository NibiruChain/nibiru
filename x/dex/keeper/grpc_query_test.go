package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/MatrixDao/matrix/x/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()

	params := types.DefaultParams()
	app.DexKeeper.SetParams(ctx, params)

	response, err := app.DexKeeper.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

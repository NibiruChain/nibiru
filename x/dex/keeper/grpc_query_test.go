package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/testutil"
	"github.com/MatrixDao/matrix/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	app, ctx := testutil.NewApp()

	params := types.DefaultParams()
	app.DexKeeper.SetParams(ctx, params)

	response, err := app.DexKeeper.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

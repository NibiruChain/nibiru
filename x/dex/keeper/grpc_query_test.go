package keeper_test

import (
	"testing"

	testkeeper "github.com/MatrixDao/matrix/testutil/keeper"
	"github.com/MatrixDao/matrix/x/dex/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	keeper, ctx, _ := testkeeper.NewDexKeeper(t, storeKey)
	wctx := sdk.WrapSDKContext(ctx)
	params := types.DefaultParams()
	keeper.SetParams(ctx, params)

	response, err := keeper.Params(wctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

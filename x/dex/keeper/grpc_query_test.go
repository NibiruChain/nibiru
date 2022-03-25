package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/testutil"
	"github.com/MatrixDao/matrix/x/dex/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	k, _, _, ctx, _ := testutil.CreateKeepers(t, storeKey)

	params := types.DefaultParams()
	k.SetParams(ctx, params)

	response, err := k.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

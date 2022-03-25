package keeper_test

import (
	"testing"

	testkeeper "github.com/MatrixDao/matrix/testutil/keeper"
	"github.com/MatrixDao/matrix/x/dex/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	k, ctx, _ := testkeeper.NewDexKeeper(t, storeKey)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

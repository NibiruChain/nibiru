package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/testutil"
	"github.com/MatrixDao/matrix/x/dex/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	k, _, _, ctx, _ := testutil.CreateKeepers(t, storeKey)

	params := types.DefaultParams()
	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

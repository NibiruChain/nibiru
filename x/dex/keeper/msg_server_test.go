package keeper_test

import (
	"context"
	"testing"

	testkeeper "github.com/MatrixDao/matrix/testutil/keeper"
	"github.com/MatrixDao/matrix/x/dex/keeper"
	"github.com/MatrixDao/matrix/x/dex/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	k, ctx, _ := testkeeper.NewDexKeeper(t, storeKey)

	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}

package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/NibiruChain/nibiru/x/poolincentives/types"
    "github.com/NibiruChain/nibiru/x/poolincentives/keeper"
    keepertest "github.com/NibiruChain/nibiru/testutil/keeper"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.PoolincentivesKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}

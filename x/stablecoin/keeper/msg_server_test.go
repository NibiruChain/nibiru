package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/MatrixDAO/dex/testutil/keeper"
	"github.com/MatrixDAO/dex/x/stablecoin/keeper"
	"github.com/MatrixDAO/dex/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.StablecoinKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}

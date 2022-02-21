package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/MatrixDao/matrix/testutil/keeper"
	"github.com/MatrixDao/matrix/x/derivatives/keeper"
	"github.com/MatrixDao/matrix/x/derivatives/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.DerivativesKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}

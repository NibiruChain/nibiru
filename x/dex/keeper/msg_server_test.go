package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/keeper"
	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/MatrixDao/matrix/x/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestCreatePool(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()

	msgServer := keeper.NewMsgServerImpl(app.DexKeeper)

	// Setup
	app.DexKeeper.SetNextPoolNumber(ctx, 1)
	addr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())

	msgCreatePool := types.MsgCreatePool{
		Creator: addr.String(),
		PoolParams: &types.PoolParams{
			SwapFee: sdk.NewDecWithPrec(3, 2),
			ExitFee: sdk.NewDecWithPrec(3, 2),
		},
		PoolAssets: []types.PoolAsset{
			{
				Token: sdk.NewCoin("uatom", sdk.NewInt(1000)),
			},
			{
				Token: sdk.NewCoin("uosmo", sdk.NewInt(1000)),
			},
		},
	}

	_, err := msgServer.CreatePool(sdk.WrapSDKContext(ctx), &msgCreatePool)
	require.Error(t, err)
	// require.EqualValues(t, resp.PoolId, 1)

}

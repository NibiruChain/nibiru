package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	"github.com/NibiruChain/nibiru/x/oracle/types"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMsgServer_EditOracleParams(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()
	goCtx := sdk.WrapSDKContext(ctx)

	msgServer := keeper.NewMsgServerImpl(app.OracleKeeper, app.SudoKeeper)

	alice := testutil.AccAddress()
	bob := testutil.AccAddress()

	// Case 1: user is not authorized to edit oracle params
	msg := types.MsgEditOracleParams{
		Sender: alice.String(),
		Params: &types.OracleParamsMsg{
			VotePeriod: 100,
		},
	}

	_, err := msgServer.EditOracleParams(goCtx, &msg)
	require.Error(t, err)
	require.EqualError(t, sudotypes.ErrUnauthorized, err.Error())

	// Case 2: user is authorized to edit oracle params
	app.SudoKeeper.Sudoers.Set(ctx, sudotypes.Sudoers{
		Root: bob.String(),
		Contracts: []string{
			alice.String(),
		},
	})

	msg = types.MsgEditOracleParams{
		Sender: alice.String(),
		Params: &types.OracleParamsMsg{
			VotePeriod: 100,
		},
	}

	_, err = msgServer.EditOracleParams(goCtx, &msg)
	require.NoError(t, err)
}

package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/inflation/keeper"
	"github.com/NibiruChain/nibiru/x/inflation/types"
)

func TestMsgToggleInflation(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()
	msgServer := keeper.NewMsgServerImpl(app.InflationKeeper)

	params := app.InflationKeeper.GetParams(ctx)
	require.False(t, params.InflationEnabled)

	msg := types.MsgToggleInflation{
		Sender: testutil.AccAddress().String(),
		Enable: false,
	}
	_, err := msgServer.ToggleInflation(ctx, &msg)
	require.ErrorContains(t, err, "insufficient permissions on smart contract")

	params = app.InflationKeeper.GetParams(ctx)
	require.False(t, params.InflationEnabled)

	msg = types.MsgToggleInflation{
		Sender: testapp.DefaultSudoRoot().String(),
		Enable: true,
	}

	_, err = msgServer.ToggleInflation(ctx, &msg)
	require.NoError(t, err)

	params = app.InflationKeeper.GetParams(ctx)
	require.True(t, params.InflationEnabled)
}

func TestMsgEditInflationParams(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()
	msgServer := keeper.NewMsgServerImpl(app.InflationKeeper)

	params := app.InflationKeeper.GetParams(ctx)
	require.NotEqualValues(t, params.EpochsPerPeriod, 42)

	newEpochPerPeriod := sdk.NewInt(42)
	msg := types.MsgEditInflationParams{
		Sender:          testutil.AccAddress().String(),
		EpochsPerPeriod: &newEpochPerPeriod,
	}
	_, err := msgServer.EditInflationParams(ctx, &msg)
	require.ErrorContains(t, err, "insufficient permissions on smart contract")

	params = app.InflationKeeper.GetParams(ctx)
	require.NotEqualValues(t, params.EpochsPerPeriod, 42)

	msg = types.MsgEditInflationParams{
		Sender:          testapp.DefaultSudoRoot().String(),
		EpochsPerPeriod: &newEpochPerPeriod,
	}

	_, err = msgServer.EditInflationParams(ctx, &msg)
	require.NoError(t, err)

	params = app.InflationKeeper.GetParams(ctx)
	require.EqualValues(t, params.EpochsPerPeriod, 42)
}

func TestMsgBurn(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()
	sender := testutil.AccAddress()
	app.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(100))))
	app.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sender, sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(100))))

	msgServer := keeper.NewMsgServerImpl(app.InflationKeeper)

	msg := types.MsgBurn{
		Sender: sender.String(),
		Coin:   sdk.NewCoin("unibi", sdk.NewInt(100)),
	}

	_, err := msgServer.Burn(ctx, &msg)
	require.NoError(t, err)
}

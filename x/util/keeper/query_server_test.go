package keeper_test

import (
	"testing"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/util/keeper"
	"github.com/NibiruChain/nibiru/x/util/types"
)

func TestQueryServer_ModuleAccounts(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(false)
	goCtx := sdktypes.WrapSDKContext(ctx)

	qServer := keeper.NewQueryServer(app.BankKeeper)

	t.Log("query accounts and check empty balance")
	accounts, err := qServer.ModuleAccounts(goCtx, &types.QueryModuleAccountsRequest{})
	require.NoError(t, err)
	require.Len(t, accounts.Accounts, len(types.ModuleAccounts))
	require.Equal(t, accounts.Accounts[0].Balance, sdktypes.Coins{})

	t.Log("we send some money")
	someModuleAccount := types.ModuleAccounts[0]
	err = app.BankKeeper.MintCoins(
		ctx,
		someModuleAccount,
		sdktypes.NewCoins(sdktypes.NewInt64Coin("uniques", 1*common.MICRO)),
	)
	require.NoError(t, err)

	t.Log("we check that it returns some balance")
	accounts, err = qServer.ModuleAccounts(goCtx, &types.QueryModuleAccountsRequest{})
	require.NoError(t, err)
	require.Len(t, accounts.Accounts, len(types.ModuleAccounts))
	require.Equal(t, accounts.Accounts[0].Balance, sdktypes.NewCoins(sdktypes.NewInt64Coin("uniques", 1*common.MICRO)))
}

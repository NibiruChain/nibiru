package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/keeper"
	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/mock"
	"github.com/MatrixDao/matrix/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestCreatePool(t *testing.T) {
	app, ctx := testutil.NewMatrixApp(true)

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
				Token:  sdk.NewCoin("uatom", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
			{
				Token:  sdk.NewCoin("uosmo", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
		},
	}

	_, err := msgServer.CreatePool(sdk.WrapSDKContext(ctx), &msgCreatePool)
	require.Error(t, err)
	// require.EqualValues(t, resp.PoolId, 1)
}

func TestJoinPool(t *testing.T) {
	const shareDenom = "matrix/pool/1"
	tests := []struct {
		name                     string
		joinerInitialFunds       sdk.Coins
		initialPool              types.Pool
		tokensIn                 sdk.Coins
		expectedNumSharesOut     sdk.Coin
		expectedRemCoins         sdk.Coins
		expectedJoinerFinalFunds sdk.Coins
		expectedFinalPool        types.Pool
	}{
		{
			name: "join with all assets",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			expectedNumSharesOut:     sdk.NewInt64Coin(shareDenom, 100),
			expectedRemCoins:         sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(sdk.NewInt64Coin(shareDenom, 100)),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 200),
					sdk.NewInt64Coin("foo", 200),
				),
				/*shares=*/ 200),
		},
		{
			name: "join with some assets, none remaining",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 50),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedRemCoins:     sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 50),
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 50),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 150),
					sdk.NewInt64Coin("foo", 150),
				),
				/*shares=*/ 150),
		},
		{
			name: "join with some assets, some remaining",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 75),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 25),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 50),
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 50),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 150),
					sdk.NewInt64Coin("foo", 150),
				),
				/*shares=*/ 150),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewMatrixApp(true)

			poolAddr := sample.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.DexKeeper.SetPool(ctx, tc.initialPool)

			joinerAddr := sample.AccAddress()
			simapp.FundAccount(app.BankKeeper, ctx, joinerAddr, tc.joinerInitialFunds)

			msgServer := keeper.NewMsgServerImpl(app.DexKeeper)
			resp, err := msgServer.JoinPool(
				sdk.WrapSDKContext(ctx),
				types.NewMsgJoinPool(joinerAddr.String(), tc.initialPool.Id, tc.tokensIn),
			)

			require.NoError(t, err)
			require.Equal(t, types.MsgJoinPoolResponse{
				Pool:             &tc.expectedFinalPool,
				NumPoolSharesOut: tc.expectedNumSharesOut,
				RemainingCoins:   tc.expectedRemCoins,
			}, *resp)
			require.Equal(t, tc.expectedJoinerFinalFunds, app.BankKeeper.GetAllBalances(ctx, joinerAddr))
		})
	}
}

func TestMsgServerExitPool(t *testing.T) {
	const shareDenom = "matrix/pool/1"
	tests := []struct {
		name                     string
		joinerInitialFunds       sdk.Coins
		initialPoolFunds         sdk.Coins
		initialPool              types.Pool
		poolSharesOut            sdk.Coin
		expectedTokensOut        sdk.Coins
		expectedJoinerFinalFunds sdk.Coins
		expectedFinalPool        types.Pool
	}{
		{
			name: "exit all pool shares",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
				sdk.NewInt64Coin(shareDenom, 100),
			),
			initialPoolFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100,
			),
			poolSharesOut: sdk.NewInt64Coin(shareDenom, 100),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 99),
				sdk.NewInt64Coin("foo", 99),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 199),
				sdk.NewInt64Coin("foo", 199),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 1),
					sdk.NewInt64Coin("foo", 1),
				),
				/*shares=*/ 0,
			),
		},
		{
			name: "exit half pool shares",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
				sdk.NewInt64Coin(shareDenom, 100),
			),
			initialPoolFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100,
			),
			poolSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 49),
				sdk.NewInt64Coin("foo", 49),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 149),
				sdk.NewInt64Coin("foo", 149),
				sdk.NewInt64Coin(shareDenom, 50),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 51),
					sdk.NewInt64Coin("foo", 51),
				),
				/*shares=*/ 50,
			),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewMatrixApp(true)

			poolAddr := sample.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.DexKeeper.SetPool(ctx, tc.initialPool)

			sender := sample.AccAddress()
			simapp.FundAccount(app.BankKeeper, ctx, sender, tc.joinerInitialFunds)
			simapp.FundAccount(app.BankKeeper, ctx, tc.initialPool.GetAddress(), tc.initialPoolFunds)

			msgServer := keeper.NewMsgServerImpl(app.DexKeeper)
			resp, err := msgServer.ExitPool(
				sdk.WrapSDKContext(ctx),
				types.NewMsgExitPool(sender.String(), tc.initialPool.Id, tc.poolSharesOut),
			)

			require.NoError(t, err)
			require.Equal(t,
				types.MsgExitPoolResponse{
					TokensOut: tc.expectedTokensOut,
				},
				*resp,
			)
			require.Equal(t,
				tc.expectedJoinerFinalFunds,
				app.BankKeeper.GetAllBalances(ctx, sender),
			)
		})
	}
}

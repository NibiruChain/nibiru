package keeper_test

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/spot/keeper"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

func TestCreatePool(t *testing.T) {
	tests := []struct {
		name               string
		creatorAddr        sdk.AccAddress
		poolParams         types.PoolParams
		poolAssets         []types.PoolAsset
		senderInitialFunds sdk.Coins
		expectedErr        error
	}{
		{
			name:        "invalid creator addr",
			creatorAddr: []byte{},
			poolParams:  types.PoolParams{PoolType: types.PoolType_BALANCER},
			poolAssets:  []types.PoolAsset{},
			expectedErr: fmt.Errorf("empty address string is not allowed"),
		},
		{
			name:        "not enough assets",
			poolParams:  types.PoolParams{PoolType: types.PoolType_BALANCER},
			poolAssets:  []types.PoolAsset{},
			expectedErr: types.ErrTooFewPoolAssets,
		},
		{
			name:       "too many assets",
			poolParams: types.PoolParams{PoolType: types.PoolType_BALANCER},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin("ccc", 1),
					Weight: sdk.OneInt(),
				},
			},
			expectedErr: types.ErrTooManyPoolAssets,
		},
		{
			name:       "asset not whitelisted 1",
			poolParams: types.PoolParams{PoolType: types.PoolType_BALANCER},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin("aaaa", 1),
					Weight: sdk.OneInt(),
				},
			},
			expectedErr: types.ErrTokenNotAllowed,
		},
		{
			name:       "asset not whitelisted 2",
			poolParams: types.PoolParams{PoolType: types.PoolType_BALANCER},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
			},
			expectedErr: types.ErrTokenNotAllowed,
		},
		{
			name:       "insufficient pool creation fee",
			poolParams: types.PoolParams{PoolType: types.PoolType_BALANCER},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.NUSD, 1),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9-1),
				sdk.NewInt64Coin("aaa", 1),
				sdk.NewInt64Coin("bbb", 1),
			),
			expectedErr: fmt.Errorf("999999999unibi is smaller than 1000000000unibi: insufficient funds"),
		},
		{
			name:       "insufficient initial deposit",
			poolParams: types.PoolParams{PoolType: types.PoolType_BALANCER},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.NIBI, 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9),
			),
			expectedErr: fmt.Errorf("0unibi is smaller than 1unibi: insufficient funds"),
		},
		{
			name:       "successful pool creation",
			poolParams: types.PoolParams{PoolType: types.PoolType_BALANCER},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.NUSD, 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9),
				sdk.NewInt64Coin(denoms.NUSD, 1),
				sdk.NewInt64Coin(denoms.USDC, 1),
			),
			expectedErr: nil,
		},
		{
			name:        "invalid creator addr - Stableswap",
			creatorAddr: []byte{},
			poolParams:  types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets:  []types.PoolAsset{},
			expectedErr: fmt.Errorf("empty address string is not allowed"),
		},
		{
			name:        "not enough assets - Stableswap",
			poolParams:  types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets:  []types.PoolAsset{},
			expectedErr: types.ErrTooFewPoolAssets,
		},
		{
			name:       "too many assets - Stableswap",
			poolParams: types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin("ccc", 1),
					Weight: sdk.OneInt(),
				},
			},
			expectedErr: types.ErrTooManyPoolAssets,
		},
		{
			name:       "asset not whitelisted 1 - Stableswap",
			poolParams: types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin("aaaa", 1),
					Weight: sdk.OneInt(),
				},
			},
			expectedErr: types.ErrTokenNotAllowed,
		},
		{
			name:       "asset not whitelisted 2 - Stableswap",
			poolParams: types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
			},
			expectedErr: types.ErrTokenNotAllowed,
		},
		{
			name:       "insufficient pool creation fee - Stableswap",
			poolParams: types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.NUSD, 1),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9-1),
				sdk.NewInt64Coin("aaa", 1),
				sdk.NewInt64Coin("bbb", 1),
			),
			expectedErr: fmt.Errorf("999999999unibi is smaller than 1000000000unibi: insufficient funds"),
		},
		{
			name:       "insufficient initial deposit - Stableswap",
			poolParams: types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.NIBI, 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9),
			),
			expectedErr: fmt.Errorf("0unibi is smaller than 1unibi: insufficient funds"),
		},
		{
			name:       "successful pool creation - Stableswap",
			poolParams: types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.NUSD, 1),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9),
				sdk.NewInt64Coin(denoms.NUSD, 1),
				sdk.NewInt64Coin(denoms.USDC, 1),
			),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.SpotKeeper)

			if tc.creatorAddr == nil {
				tc.creatorAddr = ed25519.GenPrivKey().PubKey().Address().Bytes()
			}
			if tc.senderInitialFunds != nil {
				require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, tc.creatorAddr, tc.senderInitialFunds))
			}

			msgCreatePool := types.MsgCreatePool{
				Creator:    tc.creatorAddr.String(),
				PoolParams: &tc.poolParams,
				PoolAssets: tc.poolAssets,
			}

			_, err := msgServer.CreatePool(sdk.WrapSDKContext(ctx), &msgCreatePool)
			if tc.expectedErr != nil {
				require.EqualError(t, err, tc.expectedErr.Error())
				testutil.RequireNotHasTypedEvent(t, ctx, &types.EventPoolCreated{
					Creator: tc.creatorAddr.String(),
					PoolId:  1,
				})
			} else {
				require.NoError(t, err)
				testutil.RequireHasTypedEvent(t, ctx, &types.EventPoolCreated{
					Creator: tc.creatorAddr.String(),
					PoolId:  1,
				})
			}
		})
	}
}

func TestCreateExitJoinPool(t *testing.T) {
	tests := []struct {
		name               string
		creatorAddr        sdk.AccAddress
		poolParams         types.PoolParams
		poolAssets         []types.PoolAsset
		senderInitialFunds sdk.Coins
		expectedErr        error
		useAllCoins        bool
	}{
		{
			name:       "happy path",
			poolParams: types.PoolParams{PoolType: types.PoolType_BALANCER, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.NUSD, 1_000),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1_000),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9),
				sdk.NewInt64Coin(denoms.NUSD, 1_000),
				sdk.NewInt64Coin(denoms.USDC, 1_000),
			),
			expectedErr: nil,
			useAllCoins: true,
		},
		{
			name:       "happy path - stableswap",
			poolParams: types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.NUSD, 1_000),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1_000),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9),
				sdk.NewInt64Coin(denoms.NUSD, 1_000),
				sdk.NewInt64Coin(denoms.USDC, 1_000),
			),
			expectedErr: nil,
			useAllCoins: true,
		},
		{
			name:       "happy path - no use all coins",
			poolParams: types.PoolParams{PoolType: types.PoolType_BALANCER, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.NUSD, 1_000),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1_000),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9),
				sdk.NewInt64Coin(denoms.NUSD, 1_000),
				sdk.NewInt64Coin(denoms.USDC, 1_000),
			),
			expectedErr: nil,
			useAllCoins: false,
		},
		{
			name:       "happy path - stableswap - no use all coins",
			poolParams: types.PoolParams{PoolType: types.PoolType_STABLESWAP, A: sdk.OneInt()},
			poolAssets: []types.PoolAsset{
				{
					Token:  sdk.NewInt64Coin(denoms.NUSD, 1_000),
					Weight: sdk.OneInt(),
				},
				{
					Token:  sdk.NewInt64Coin(denoms.USDC, 1_000),
					Weight: sdk.OneInt(),
				},
			},
			senderInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1e9),
				sdk.NewInt64Coin(denoms.NUSD, 1_000),
				sdk.NewInt64Coin(denoms.USDC, 1_000),
			),
			expectedErr: nil,
			useAllCoins: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.SpotKeeper)

			if tc.creatorAddr == nil {
				tc.creatorAddr = ed25519.GenPrivKey().PubKey().Address().Bytes()
			}
			if tc.senderInitialFunds != nil {
				require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, tc.creatorAddr, tc.senderInitialFunds))
			}

			msgCreatePool := types.MsgCreatePool{
				Creator:    tc.creatorAddr.String(),
				PoolParams: &tc.poolParams,
				PoolAssets: tc.poolAssets,
			}

			_, err := msgServer.CreatePool(sdk.WrapSDKContext(ctx), &msgCreatePool)
			require.NoError(t, err)
			testutil.RequireHasTypedEvent(t, ctx, &types.EventPoolCreated{
				Creator: tc.creatorAddr.String(),
				PoolId:  1,
			})

			poolShares := app.BankKeeper.GetBalance(ctx, tc.creatorAddr, "nibiru/pool/1")
			msgExitPool := types.MsgExitPool{
				Sender:     tc.creatorAddr.String(),
				PoolId:     1,
				PoolShares: poolShares,
			}
			_, err = msgServer.ExitPool(sdk.WrapSDKContext(ctx), &msgExitPool)
			require.NoError(t, err)

			require.Equal(
				t,
				tc.senderInitialFunds.Sub(sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1e9))),
				app.BankKeeper.GetAllBalances(ctx, tc.creatorAddr),
			)

			msgJoinPool := types.MsgJoinPool{
				Sender:      tc.creatorAddr.String(),
				PoolId:      1,
				TokensIn:    tc.senderInitialFunds.Sub(sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1e9))),
				UseAllCoins: tc.useAllCoins,
			}
			_, err = msgServer.JoinPool(sdk.WrapSDKContext(ctx), &msgJoinPool)
			require.NoError(t, err)

			require.Equal(
				t,
				sdk.NewCoins(poolShares),
				app.BankKeeper.GetAllBalances(ctx, tc.creatorAddr),
			)
		})
	}
}

func TestMsgServerJoinPool(t *testing.T) {
	const shareDenom = "nibiru/pool/1"
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
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			expectedNumSharesOut:     sdk.NewInt64Coin(shareDenom, 100),
			expectedRemCoins:         sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(sdk.NewInt64Coin(shareDenom, 100)),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 200),
					sdk.NewInt64Coin(denoms.NUSD, 200),
				),
				/*shares=*/ 200),
		},
		{
			name: "join with some assets, none remaining",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 50),
				sdk.NewInt64Coin(denoms.NUSD, 50),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedRemCoins:     sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 50),
				sdk.NewInt64Coin(denoms.NIBI, 50),
				sdk.NewInt64Coin(denoms.NUSD, 50),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 150),
					sdk.NewInt64Coin(denoms.NUSD, 150),
				),
				/*shares=*/ 150),
		},
		{
			name: "join with some assets, some remaining",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 50),
				sdk.NewInt64Coin(denoms.NUSD, 75),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NUSD, 25),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 50),
				sdk.NewInt64Coin(denoms.NIBI, 50),
				sdk.NewInt64Coin(denoms.NUSD, 50),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 150),
					sdk.NewInt64Coin(denoms.NUSD, 150),
				),
				/*shares=*/ 150),
		},
		{
			name: "join with all assets - Stablepool",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			expectedNumSharesOut:     sdk.NewInt64Coin(shareDenom, 100),
			expectedRemCoins:         sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(sdk.NewInt64Coin(shareDenom, 100)),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 200),
					sdk.NewInt64Coin(denoms.NUSD, 200),
				),
				/*shares=*/ 200),
		},
		{
			name: "join with some assets, none remaining - Stablepool",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 50),
				sdk.NewInt64Coin(denoms.NUSD, 50),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedRemCoins:     []sdk.Coin{},
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 50),
				sdk.NewInt64Coin(denoms.NIBI, 50),
				sdk.NewInt64Coin(denoms.NUSD, 50),
			),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 150),
					sdk.NewInt64Coin(denoms.NUSD, 150),
				),
				/*shares=*/ 150),
		},
		{
			name: "join with some assets, some remaining - Stablepool",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 50),
				sdk.NewInt64Coin(denoms.NUSD, 75),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 62),
			expectedRemCoins:     []sdk.Coin{},
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 62),
				sdk.NewInt64Coin(denoms.NIBI, 50),
				sdk.NewInt64Coin(denoms.NUSD, 25),
			),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 150),
					sdk.NewInt64Coin(denoms.NUSD, 175),
				),
				/*shares=*/ 162),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			poolAddr := testutil.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.SpotKeeper.SetPool(ctx, tc.initialPool)

			joinerAddr := testutil.AccAddress()
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, joinerAddr, tc.joinerInitialFunds))

			msgServer := keeper.NewMsgServerImpl(app.SpotKeeper)
			resp, err := msgServer.JoinPool(
				sdk.WrapSDKContext(ctx),
				types.NewMsgJoinPool(joinerAddr.String(), tc.initialPool.Id, tc.tokensIn, false),
			)

			require.NoError(t, err)
			require.Equal(t, types.MsgJoinPoolResponse{
				Pool:             &tc.expectedFinalPool,
				NumPoolSharesOut: tc.expectedNumSharesOut,
				RemainingCoins:   tc.expectedRemCoins,
			}, *resp)
			require.Equal(t, tc.expectedJoinerFinalFunds, app.BankKeeper.GetAllBalances(ctx, joinerAddr))
			expectedEvent := &types.EventPoolJoined{
				Address:       joinerAddr.String(),
				PoolId:        1,
				TokensIn:      tc.tokensIn,
				PoolSharesOut: resp.NumPoolSharesOut,
				RemCoins:      resp.RemainingCoins,
			}
			testutil.RequireHasTypedEvent(t, ctx, expectedEvent)
		})
	}
}

func TestMsgServerExitPool(t *testing.T) {
	const shareDenom = "nibiru/pool/1"
	tests := []struct {
		name                     string
		joinerInitialFunds       sdk.Coins
		poolFundsToAdd           sdk.Coins
		initialPool              types.Pool
		poolSharesIn             sdk.Coin
		expectedTokensOut        sdk.Coins
		expectedJoinerFinalFunds sdk.Coins
		expectedFinalPool        types.Pool
	}{
		{
			name: "exit all pool shares",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
				sdk.NewInt64Coin(shareDenom, 100),
			),
			poolFundsToAdd: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			poolSharesIn: sdk.NewInt64Coin(shareDenom, 100),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 99),
				sdk.NewInt64Coin(denoms.NUSD, 99),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 199),
				sdk.NewInt64Coin(denoms.NUSD, 199),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 1),
					sdk.NewInt64Coin(denoms.NUSD, 1),
				),
				/*shares=*/ 0,
			),
		},
		{
			name: "exit half pool shares",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
				sdk.NewInt64Coin(shareDenom, 100),
			),
			poolFundsToAdd: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			poolSharesIn: sdk.NewInt64Coin(shareDenom, 50),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 49),
				sdk.NewInt64Coin(denoms.NUSD, 49),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 149),
				sdk.NewInt64Coin(denoms.NUSD, 149),
				sdk.NewInt64Coin(shareDenom, 50),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 51),
					sdk.NewInt64Coin(denoms.NUSD, 51),
				),
				/*shares=*/ 50,
			),
		},
		{
			name: "exit all pool shares - StablePool",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
				sdk.NewInt64Coin(shareDenom, 100),
			),
			poolFundsToAdd: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			poolSharesIn: sdk.NewInt64Coin(shareDenom, 100),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 200),
				sdk.NewInt64Coin(denoms.NUSD, 200),
			),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 1),
					sdk.NewInt64Coin(denoms.NUSD, 1),
				),
				/*shares=*/ 0,
			),
		},
		{
			name: "exit half pool shares - StablePool",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
				sdk.NewInt64Coin(shareDenom, 100),
			),
			poolFundsToAdd: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			poolSharesIn: sdk.NewInt64Coin(shareDenom, 50),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 50),
				sdk.NewInt64Coin(denoms.NUSD, 50),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 150),
				sdk.NewInt64Coin(denoms.NUSD, 150),
				sdk.NewInt64Coin(shareDenom, 50),
			),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 50),
					sdk.NewInt64Coin(denoms.NUSD, 50),
				),
				/*shares=*/ 50,
			),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			poolAddr := testutil.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.SpotKeeper.SetPool(ctx, tc.initialPool)

			sender := testutil.AccAddress()
			require.NoError(t, simapp.FundAccount(
				app.BankKeeper, ctx, sender, tc.joinerInitialFunds))
			require.NoError(t, simapp.FundAccount(
				app.BankKeeper, ctx, tc.initialPool.GetAddress(), tc.poolFundsToAdd))

			msgServer := keeper.NewMsgServerImpl(app.SpotKeeper)
			resp, err := msgServer.ExitPool(
				sdk.WrapSDKContext(ctx),
				types.NewMsgExitPool(sender.String(), tc.initialPool.Id, tc.poolSharesIn),
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

			expectedEvent := &types.EventPoolExited{
				Address:      sender.String(),
				PoolId:       1,
				PoolSharesIn: tc.poolSharesIn,
				TokensOut:    resp.TokensOut,
			}

			testutil.RequireHasTypedEvent(t, ctx, expectedEvent)
		})
	}
}

func TestMsgServerSwapAssets(t *testing.T) {
	tests := []struct {
		name string

		// test setup
		userInitialFunds sdk.Coins
		initialPool      types.Pool
		tokenIn          sdk.Coin
		tokenOutDenom    string

		// expected results
		expectedError          error
		expectedTokenOut       sdk.Coin
		expectedUserFinalFunds sdk.Coins
		expectedFinalPool      types.Pool
	}{
		{
			name: "regular swap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:          sdk.NewInt64Coin(denoms.NIBI, 100),
			tokenOutDenom:    denoms.NUSD,
			expectedTokenOut: sdk.NewInt64Coin(denoms.NUSD, 50),
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NUSD, 50),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 200),
					sdk.NewInt64Coin(denoms.NUSD, 50),
				),
				/*shares=*/ 100,
			),
			expectedError: nil,
		},
		{
			name: "not enough user funds",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin(denoms.NIBI, 100),
			tokenOutDenom: denoms.NUSD,
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			expectedError: sdkerrors.ErrInsufficientFunds,
		},
		{
			name: "invalid token in denom",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin("foo", 100),
			tokenOutDenom: denoms.NUSD,
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 100),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrTokenDenomNotFound,
		},
		{
			name: "invalid token out denom",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin(denoms.NIBI, 100),
			tokenOutDenom: "foo",
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrTokenDenomNotFound,
		},
		{
			name: "same token in and token out denom",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin(denoms.NIBI, 100),
			tokenOutDenom: denoms.NIBI,
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrSameTokenDenom,
		},
		{
			name: "regular swap - StableSwap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:          sdk.NewInt64Coin(denoms.NIBI, 100),
			tokenOutDenom:    denoms.NUSD,
			expectedTokenOut: sdk.NewInt64Coin(denoms.NUSD, 95),
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NUSD, 95),
			),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 200),
					sdk.NewInt64Coin(denoms.NUSD, 5),
				),
				/*shares=*/ 100,
			),
			expectedError: nil,
		},
		{
			name: "not enough user funds - StableSwap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin(denoms.NIBI, 100),
			tokenOutDenom: denoms.NUSD,
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 1),
			),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			expectedError: sdkerrors.ErrInsufficientFunds,
		},
		{
			name: "invalid token in denom - StableSwap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin("foo", 100),
			tokenOutDenom: denoms.NUSD,
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 100),
			),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrTokenDenomNotFound,
		},
		{
			name: "invalid token out denom - StableSwap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin(denoms.NIBI, 100),
			tokenOutDenom: "foo",
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrTokenDenomNotFound,
		},
		{
			name: "same token in and token out denom - StableSwap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			initialPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin(denoms.NIBI, 100),
			tokenOutDenom: denoms.NIBI,
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NIBI, 100),
			),
			expectedFinalPool: mock.SpotStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrSameTokenDenom,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.SpotKeeper)

			// fund pool account
			poolAddr := testutil.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			require.NoError(t,
				simapp.FundAccount(
					app.BankKeeper,
					ctx,
					poolAddr,
					tc.initialPool.PoolBalances(),
				),
			)
			app.SpotKeeper.SetPool(ctx, tc.initialPool)

			// fund user account
			sender := testutil.AccAddress()
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, sender, tc.userInitialFunds))

			// swap assets
			resp, err := msgServer.SwapAssets(
				sdk.WrapSDKContext(ctx),
				types.NewMsgSwapAssets(sender.String(), tc.initialPool.Id, tc.tokenIn, tc.tokenOutDenom),
			)

			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					types.MsgSwapAssetsResponse{
						TokenOut: tc.expectedTokenOut,
					},
					*resp,
				)

				// check events
				testutil.RequireHasTypedEvent(t, ctx, &types.EventAssetsSwapped{
					Address:  sender.String(),
					PoolId:   1,
					TokenIn:  tc.tokenIn,
					TokenOut: tc.expectedTokenOut,
					Fee:      sdk.NewInt64Coin("unibi", 0),
				})
			}

			// check user's final funds
			require.Equal(t,
				tc.expectedUserFinalFunds,
				app.BankKeeper.GetAllBalances(ctx, sender),
			)

			// check final pool state
			finalPool, err := app.SpotKeeper.FetchPool(ctx, tc.initialPool.Id)
			require.NoError(t, err)
			require.Equal(t, tc.expectedFinalPool, finalPool)
		})
	}
}

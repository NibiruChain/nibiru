package keeper_test

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	devgas "github.com/NibiruChain/nibiru/v2/x/devgas/v1"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

// TestClearFeeShareState checks that the shutdown cleanup deletes every fee
// share and both of its indexes, disables fee sharing while keeping the other
// params, leaves a genesis that still validates, and is safe to run twice.
func (s *KeeperTestSuite) TestClearFeeShareState() {
	s.SetupTest()
	k := s.app.DevGasKeeper

	deployer := testutil.NewAccAddress().String()
	withdrawer := testutil.NewAccAddress().String()
	const numFeeShares = 5
	for range numFeeShares {
		k.SetFeeShare(s.ctx, devgastypes.FeeShare{
			ContractAddress:   testutil.NewAccAddress().String(),
			DeployerAddress:   deployer,
			WithdrawerAddress: withdrawer,
		})
	}
	// One fee share with a different deployer and withdrawer.
	k.SetFeeShare(s.ctx, devgastypes.FeeShare{
		ContractAddress:   testutil.NewAccAddress().String(),
		DeployerAddress:   testutil.NewAccAddress().String(),
		WithdrawerAddress: testutil.NewAccAddress().String(),
	})

	paramsBefore := k.GetParams(s.ctx)
	s.Require().True(paramsBefore.EnableFeeShare)
	s.Require().Len(devgas.ExportGenesis(s.ctx, k).FeeShare, numFeeShares+1)

	deleted, err := k.ClearFeeShareState(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(numFeeShares+1, deleted)

	s.T().Log("no fee shares remain in the primary store or the indexes")
	goCtx := sdk.WrapSDKContext(s.ctx)
	byDeployer, err := s.queryClient.FeeShares(
		goCtx, &devgastypes.QueryFeeSharesRequest{Deployer: deployer},
	)
	s.Require().NoError(err)
	s.Require().Empty(byDeployer.Feeshare)
	byWithdrawer, err := s.queryClient.FeeSharesByWithdrawer(
		goCtx, &devgastypes.QueryFeeSharesByWithdrawerRequest{
			WithdrawerAddress: withdrawer,
		},
	)
	s.Require().NoError(err)
	s.Require().Empty(byWithdrawer.Feeshare)
	store := s.ctx.KVStore(s.app.GetKey(devgastypes.StoreKey))
	for _, prefix := range [][]byte{
		devgastypes.KeyPrefixFeeShare.Prefix(),
		devgastypes.KeyPrefixDeployer.Prefix(),
		devgastypes.KeyPrefixWithdrawer.Prefix(),
	} {
		iter := sdk.KVStorePrefixIterator(store, prefix)
		s.Require().Falsef(iter.Valid(), "prefix %v still has keys", prefix)
		s.Require().NoError(iter.Close())
	}

	s.T().Log("fee sharing is disabled and the other params are unchanged")
	wantParams := paramsBefore
	wantParams.EnableFeeShare = false
	s.Require().Equal(wantParams, k.GetParams(s.ctx))

	s.T().Log("the exported genesis is empty and valid")
	genesis := devgas.ExportGenesis(s.ctx, k)
	s.Require().Empty(genesis.FeeShare)
	s.Require().NoError(genesis.Validate())

	s.T().Log("a second run deletes nothing")
	deleted, err = k.ClearFeeShareState(s.ctx)
	s.Require().NoError(err)
	s.Require().Zero(deleted)
	s.Require().Equal(wantParams, k.GetParams(s.ctx))
}

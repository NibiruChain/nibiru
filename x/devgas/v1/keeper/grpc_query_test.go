package keeper_test

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	sdkmath "cosmossdk.io/math"

	devgaskeeper "github.com/NibiruChain/nibiru/v2/x/devgas/v1/keeper"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

func (s *KeeperTestSuite) TestQueryFeeShares() {
	s.SetupTest()
	_, _, sender := testdata.KeyTestPubAddr()
	_ = s.FundAccount(
		s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(1_000_000))),
	)

	_, _, withdrawer := testdata.KeyTestPubAddr()

	var contractAddressList []string
	var index uint64
	for index < 5 {
		contractAddress := s.InstantiateContract(sender.String(), "")
		contractAddressList = append(contractAddressList, contractAddress)
		index++
	}

	// Register FeeShares
	var feeShares []devgastypes.FeeShare
	for _, contractAddress := range contractAddressList {
		feeShare := devgastypes.FeeShare{
			ContractAddress:   contractAddress,
			DeployerAddress:   sender.String(),
			WithdrawerAddress: withdrawer.String(),
		}

		feeShares = append(feeShares, feeShare)
		s.app.DevGasKeeper.SetFeeShare(s.ctx, feeShare)
	}

	s.Run("from deployer", func() {
		deployer := sender.String()
		req := &devgastypes.QueryFeeSharesRequest{
			Deployer: deployer,
		}
		goCtx := sdk.WrapSDKContext(s.ctx)
		resp, err := s.queryClient.FeeShares(goCtx, req)
		s.NoError(err)
		s.Len(resp.Feeshare, len(feeShares))
	})
	s.Run("from random", func() {
		deployer := testutil.NewAccAddress().String()
		req := &devgastypes.QueryFeeSharesRequest{
			Deployer: deployer,
		}
		goCtx := sdk.WrapSDKContext(s.ctx)
		resp, err := s.queryClient.FeeShares(goCtx, req)
		s.NoError(err)
		s.Len(resp.Feeshare, 0)
	})
}

func (s *KeeperTestSuite) TestFeeShare() {
	s.SetupTest()
	_, _, sender := testdata.KeyTestPubAddr()
	_ = s.FundAccount(s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(1_000_000))))

	_, _, withdrawer := testdata.KeyTestPubAddr()

	contractAddress := s.InstantiateContract(sender.String(), "")
	feeShare := devgastypes.FeeShare{
		ContractAddress:   contractAddress,
		DeployerAddress:   sender.String(),
		WithdrawerAddress: withdrawer.String(),
	}
	s.app.DevGasKeeper.SetFeeShare(s.ctx, feeShare)

	req := &devgastypes.QueryFeeShareRequest{
		ContractAddress: contractAddress,
	}
	goCtx := sdk.WrapSDKContext(s.ctx)
	resp, err := s.queryClient.FeeShare(goCtx, req)
	s.Require().NoError(err)
	s.Require().Equal(resp.Feeshare, feeShare)
}

func (s *KeeperTestSuite) TestFeeSharesByWithdrawer() {
	s.SetupTest()
	_, _, sender := testdata.KeyTestPubAddr()
	_ = s.FundAccount(s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(1_000_000))))

	_, _, withdrawer := testdata.KeyTestPubAddr()

	var contractAddressList []string
	var index uint64
	for index < 5 {
		contractAddress := s.InstantiateContract(sender.String(), "")
		contractAddressList = append(contractAddressList, contractAddress)
		index++
	}

	for _, contractAddress := range contractAddressList {
		s.app.DevGasKeeper.SetFeeShare(s.ctx, devgastypes.FeeShare{
			ContractAddress:   contractAddress,
			DeployerAddress:   sender.String(),
			WithdrawerAddress: withdrawer.String(),
		})
	}

	s.Run("Total", func() {
		goCtx := sdk.WrapSDKContext(s.ctx)
		resp, err := s.queryClient.FeeSharesByWithdrawer(goCtx,
			&devgastypes.QueryFeeSharesByWithdrawerRequest{
				WithdrawerAddress: withdrawer.String(),
			})
		s.Require().NoError(err)
		s.Require().Equal(len(contractAddressList), len(resp.Feeshare))
	})
}

func (s *KeeperTestSuite) TestQueryParams() {
	s.SetupTest()
	goCtx := sdk.WrapSDKContext(s.ctx)
	resp, err := s.queryClient.Params(goCtx, nil)
	s.NoError(err)
	s.NotNil(resp)
}

func (s *KeeperTestSuite) TestNilRequests() {
	s.SetupTest()
	goCtx := sdk.WrapSDKContext(s.ctx)
	querier := devgaskeeper.NewQuerier(s.app.DevGasKeeper)

	_, err := querier.FeeShare(goCtx, nil)
	s.Error(err)

	_, err = querier.FeeShares(goCtx, nil)
	s.Error(err)

	_, err = querier.FeeSharesByWithdrawer(goCtx, nil)
	s.Error(err)
}

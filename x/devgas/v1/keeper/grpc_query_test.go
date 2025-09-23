package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	devgaskeeper "github.com/NibiruChain/nibiru/v2/x/devgas/v1/keeper"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
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
		msg := &devgastypes.MsgRegisterFeeShare{
			ContractAddress:   contractAddress,
			DeployerAddress:   sender.String(),
			WithdrawerAddress: withdrawer.String(),
		}

		feeShare := devgastypes.FeeShare{
			ContractAddress:   contractAddress,
			DeployerAddress:   sender.String(),
			WithdrawerAddress: withdrawer.String(),
		}

		feeShares = append(feeShares, feeShare)

		_, err := s.devgasMsgServer.RegisterFeeShare(s.ctx, msg)
		s.Require().NoError(err)
	}

	s.Run("from deployer", func() {
		deployer := sender.String()
		req := &devgastypes.QueryFeeSharesRequest{
			Deployer: deployer,
		}

		resp, err := s.queryClient.FeeShares(s.ctx, req)
		s.NoError(err)
		s.Len(resp.Feeshare, len(feeShares))
	})
	s.Run("from random", func() {
		deployer := testutil.AccAddress().String()
		req := &devgastypes.QueryFeeSharesRequest{
			Deployer: deployer,
		}

		resp, err := s.queryClient.FeeShares(s.ctx, req)
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

	msg := &devgastypes.MsgRegisterFeeShare{
		ContractAddress:   contractAddress,
		DeployerAddress:   sender.String(),
		WithdrawerAddress: withdrawer.String(),
	}

	feeShare := devgastypes.FeeShare{
		ContractAddress:   contractAddress,
		DeployerAddress:   sender.String(),
		WithdrawerAddress: withdrawer.String(),
	}
	_, err := s.devgasMsgServer.RegisterFeeShare(s.ctx, msg)
	s.Require().NoError(err)

	req := &devgastypes.QueryFeeShareRequest{
		ContractAddress: contractAddress,
	}
	resp, err := s.queryClient.FeeShare(s.ctx, req)
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

	// RegsisFeeShare
	for _, contractAddress := range contractAddressList {
		msg := &devgastypes.MsgRegisterFeeShare{
			ContractAddress:   contractAddress,
			DeployerAddress:   sender.String(),
			WithdrawerAddress: withdrawer.String(),
		}

		_, err := s.devgasMsgServer.RegisterFeeShare(s.ctx, msg)
		s.Require().NoError(err)
	}

	s.Run("Total", func() {
		resp, err := s.queryClient.FeeSharesByWithdrawer(s.ctx,
			&devgastypes.QueryFeeSharesByWithdrawerRequest{
				WithdrawerAddress: withdrawer.String(),
			})
		s.Require().NoError(err)
		s.Require().Equal(len(contractAddressList), len(resp.Feeshare))
	})
}

func (s *KeeperTestSuite) TestQueryParams() {
	s.SetupTest()

	resp, err := s.queryClient.Params(s.ctx, nil)
	s.NoError(err)
	s.NotNil(resp)
}

func (s *KeeperTestSuite) TestNilRequests() {
	s.SetupTest()

	querier := devgaskeeper.NewQuerier(s.app.DevGasKeeper)

	_, err := querier.FeeShare(s.ctx, nil)
	s.Error(err)

	_, err = querier.FeeShares(s.ctx, nil)
	s.Error(err)

	_, err = querier.FeeSharesByWithdrawer(s.ctx, nil)
	s.Error(err)
}

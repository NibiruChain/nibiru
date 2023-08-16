package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

func (s *IntegrationTestSuite) TestQueryFeeShares() {
	s.SetupTest()
	_, _, sender := testdata.KeyTestPubAddr()
	_ = s.FundAccount(
		s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1_000_000))),
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
	var feeShares []types.FeeShare
	for _, contractAddress := range contractAddressList {
		goCtx := sdk.WrapSDKContext(s.ctx)
		msg := &types.MsgRegisterFeeShare{
			ContractAddress:   contractAddress,
			DeployerAddress:   sender.String(),
			WithdrawerAddress: withdrawer.String(),
		}

		feeShare := types.FeeShare{
			ContractAddress:   contractAddress,
			DeployerAddress:   sender.String(),
			WithdrawerAddress: withdrawer.String(),
		}

		feeShares = append(feeShares, feeShare)

		_, err := s.devgasMsgServer.RegisterFeeShare(goCtx, msg)
		s.Require().NoError(err)
	}

	s.Run("from deployer", func() {
		deployer := sender.String()
		req := &types.QueryFeeSharesRequest{
			Deployer: deployer,
		}
		goCtx := sdk.WrapSDKContext(s.ctx)
		resp, err := s.queryClient.FeeShares(goCtx, req)
		s.NoError(err)
		s.Len(resp.Feeshare, len(feeShares))
	})
	s.Run("from random", func() {
		deployer := testutil.AccAddress().String()
		req := &types.QueryFeeSharesRequest{
			Deployer: deployer,
		}
		goCtx := sdk.WrapSDKContext(s.ctx)
		resp, err := s.queryClient.FeeShares(goCtx, req)
		s.NoError(err)
		s.Len(resp.Feeshare, 0)
	})
}

func (s *IntegrationTestSuite) TestFeeShare() {
	s.SetupTest()
	_, _, sender := testdata.KeyTestPubAddr()
	_ = s.FundAccount(s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1_000_000))))

	_, _, withdrawer := testdata.KeyTestPubAddr()

	contractAddress := s.InstantiateContract(sender.String(), "")
	goCtx := sdk.WrapSDKContext(s.ctx)
	msg := &types.MsgRegisterFeeShare{
		ContractAddress:   contractAddress,
		DeployerAddress:   sender.String(),
		WithdrawerAddress: withdrawer.String(),
	}

	feeShare := types.FeeShare{
		ContractAddress:   contractAddress,
		DeployerAddress:   sender.String(),
		WithdrawerAddress: withdrawer.String(),
	}
	_, err := s.devgasMsgServer.RegisterFeeShare(goCtx, msg)
	s.Require().NoError(err)

	req := &types.QueryFeeShareRequest{
		ContractAddress: contractAddress,
	}
	goCtx = sdk.WrapSDKContext(s.ctx)
	resp, err := s.queryClient.FeeShare(goCtx, req)
	s.Require().NoError(err)
	s.Require().Equal(resp.Feeshare, feeShare)
}

func (s *IntegrationTestSuite) TestFeeSharesByWithdrawer() {
	s.SetupTest()
	_, _, sender := testdata.KeyTestPubAddr()
	_ = s.FundAccount(s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1_000_000))))

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
		goCtx := sdk.WrapSDKContext(s.ctx)
		msg := &types.MsgRegisterFeeShare{
			ContractAddress:   contractAddress,
			DeployerAddress:   sender.String(),
			WithdrawerAddress: withdrawer.String(),
		}

		_, err := s.devgasMsgServer.RegisterFeeShare(goCtx, msg)
		s.Require().NoError(err)
	}

	s.Run("Total", func() {
		goCtx := sdk.WrapSDKContext(s.ctx)
		resp, err := s.queryClient.FeeSharesByWithdrawer(goCtx,
			&types.QueryFeeSharesByWithdrawerRequest{
				WithdrawerAddress: withdrawer.String(),
			})
		s.Require().NoError(err)
		s.Require().Equal(len(contractAddressList), len(resp.Feeshare))
	})
}

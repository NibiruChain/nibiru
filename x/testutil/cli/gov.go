package cli

import (
	"context"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/suite"
	abcitypes "github.com/tendermint/tendermint/abci/types"
)

func PassGovProposal(s suite.Suite, network *Network) {
	val := network.Validators[0]
	clientCtx := val.ClientCtx.WithOutputFormat("json")
	govQueryClient := govtypes.NewQueryClient(clientCtx)
	s.NoError(network.WaitForNextBlock())

	// ----------------------------------------------------------------------
	s.T().Log(`Check that proposal was correctly submitted with gov client
	$ nibid query gov proposal 1`)
	// ----------------------------------------------------------------------
	proposalsQueryResp, err := govQueryClient.Proposals(
		context.Background(), &govtypes.QueryProposalsRequest{
			Depositor: val.Address.String(),
		},
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(proposalsQueryResp.Proposals)

	s.Equalf(
		govtypes.StatusDepositPeriod,
		proposalsQueryResp.Proposals[0].Status,
		"proposal should be in deposit period as it hasn't passed min deposit")
	s.EqualValues(
		sdk.NewCoins(sdk.NewInt64Coin("unibi", 1000)),
		proposalsQueryResp.Proposals[0].TotalDeposit,
	)

	// ----------------------------------------------------------------------
	s.T().Log(`Move proposal to vote status by meeting min deposit
	$ nibid tx gov deposit [proposal-id] [deposit] [flags]`)
	// ----------------------------------------------------------------------
	govDepositParams, err := govQueryClient.Params(
		context.Background(),
		&govtypes.QueryParamsRequest{
			ParamsType: govtypes.ParamDeposit,
		},
	)
	s.NoError(err)

	proposalId := proposalsQueryResp.Proposals[0].ProposalId
	args := []string{
		/*proposal-id=*/ strconv.Itoa(int(proposalId)),
		/*deposit=*/ govDepositParams.DepositParams.MinDeposit.String(),
	}
	txResp, err := ExecTx(network, govcli.NewCmdDeposit(), val.Address, args)
	s.Require().NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	proposalQueryResponse, err := govQueryClient.Proposal(
		context.Background(),
		&govtypes.QueryProposalRequest{
			ProposalId: proposalId,
		},
	)
	s.Require().NoError(err)
	s.Equalf(
		govtypes.StatusVotingPeriod,
		proposalQueryResponse.Proposal.Status,
		"proposal should be in voting period since min deposit has been met",
	)

	// ----------------------------------------------------------------------
	s.T().Log(`Vote on the proposal.
	$ nibid tx gov vote [proposal-id] [option] [flags]
	e.g. $ nibid tx gov vote 1 yes`)
	// ----------------------------------------------------------------------
	args = []string{
		/*proposal-id=*/ strconv.Itoa(int(proposalId)),
		/*option=*/ "yes",
	}

	txResp, err = ExecTx(network, govcli.NewCmdVote(), val.Address, args)
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.Require().Eventuallyf(func() bool {
		proposalQueryResp, err := govQueryClient.Proposal(
			context.Background(), &govtypes.QueryProposalRequest{ProposalId: proposalId})
		s.Require().NoError(err)
		return govtypes.StatusPassed == proposalQueryResp.Proposal.Status
	}, 20*time.Second, 2*time.Second,
		"proposal should pass after voting period")
}

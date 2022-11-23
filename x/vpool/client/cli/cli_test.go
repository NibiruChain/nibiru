package cli_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/suite"
	abcitypes "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	app.SetPrefixes(app.AccountAddressPrefix)

	encodingConfig := simapp.MakeTestEncodingConfig()
	genesisState := simapp.NewTestGenesisStateFromDefault()
	vpoolGenesis := vpooltypes.DefaultGenesis()
	vpoolGenesis.Vpools = []vpooltypes.Vpool{
		{
			Pair:              common.Pair_ETH_NUSD,
			BaseAssetReserve:  sdk.NewDec(10_000_000),
			QuoteAssetReserve: sdk.NewDec(60_000_000_000),
			Config: vpooltypes.VpoolConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
		},
	}
	genesisState[vpooltypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(vpoolGenesis)

	s.cfg = testutilcli.BuildNetworkConfig(genesisState)
	s.network = testutilcli.NewNetwork(s.T(), s.cfg)
	s.Require().NoError(s.network.WaitForNextBlock())
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestCmdCreatePoolProposal() {
	val := s.network.Validators[0]
	clientCtx := val.ClientCtx.WithOutputFormat("json")
	govQueryClient := govtypes.NewQueryClient(clientCtx)

	// ----------------------------------------------------------------------
	s.T().Log("load example proposal json as bytes")
	// ----------------------------------------------------------------------
	proposal := &vpooltypes.CreatePoolProposal{
		Title:             "Create ETH:USD pool",
		Description:       "Creates an ETH:USD pool",
		Pair:              "ETH:USD",
		QuoteAssetReserve: sdk.NewDec(1_000_000),
		BaseAssetReserve:  sdk.NewDec(1_000_000),
		Config: vpooltypes.VpoolConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.05"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.05"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	}
	proposalFile := sdktestutil.WriteToNewTempFile(s.T(), string(val.ClientCtx.Codec.MustMarshalJSON(proposal)))
	contents, err := os.ReadFile(proposalFile.Name())
	s.Require().NoError(err)

	s.T().Log("Unmarshal json bytes into proposal object; check validity")
	proposal = &vpooltypes.CreatePoolProposal{}
	val.ClientCtx.Codec.MustUnmarshalJSON(contents, proposal)
	s.Require().NoError(proposal.ValidateBasic())

	// ----------------------------------------------------------------------
	s.T().Log("Submit proposal and unmarshal tx response")
	// ----------------------------------------------------------------------
	args := []string{
		proposalFile.Name(),
		fmt.Sprintf("--%s=1000unibi", govcli.FlagDeposit),
	}
	cmd := cli.CmdCreatePoolProposal()
	flags.AddTxFlagsToCmd(cmd)
	txResp, err := testutilcli.ExecTx(s.network, cmd, val.Address, args)
	s.Require().NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

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
	args = []string{
		/*proposal-id=*/ strconv.Itoa(int(proposalId)),
		/*deposit=*/ govDepositParams.DepositParams.MinDeposit.String(),
	}
	txResp, err = testutilcli.ExecTx(s.network, govcli.NewCmdDeposit(), val.Address, args)
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

	txResp, err = testutilcli.ExecTx(s.network, govcli.NewCmdVote(), val.Address, args)
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.Require().Eventuallyf(
		func() bool {
			proposalQueryResp, err := govQueryClient.Proposal(
				context.Background(), &govtypes.QueryProposalRequest{ProposalId: proposalId})
			s.Require().NoError(err)
			return govtypes.StatusPassed == proposalQueryResp.Proposal.Status
		},
		20*time.Second,
		2*time.Second,
		"proposal should pass after voting period")

	// ----------------------------------------------------------------------
	s.T().Log("verify that the new proposed pool exists")
	// ----------------------------------------------------------------------
	s.Require().NoError(s.network.WaitForNextBlock())

	vpoolsQueryResp := &vpooltypes.QueryAllPoolsResponse{}
	s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdGetVpools(), nil, vpoolsQueryResp))

	found := false
	for _, pool := range vpoolsQueryResp.Pools {
		if pool.Pair.String() == proposal.Pair {
			s.EqualValues(vpooltypes.Vpool{
				Pair:              common.MustNewAssetPair(proposal.Pair),
				BaseAssetReserve:  proposal.BaseAssetReserve,
				QuoteAssetReserve: proposal.QuoteAssetReserve,
				Config:            proposal.Config,
			}, pool)
			found = true
		}
	}
	s.True(found, "pool does not exist")
}

func (s *IntegrationTestSuite) TestGetPrices() {
	val := s.network.Validators[0]

	s.T().Log("check vpool balances")
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.Pair_ETH_NUSD)
	s.NoError(err)
	s.EqualValues(sdk.MustNewDecFromStr("10000000"), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.MustNewDecFromStr("60000000000"), reserveAssets.QuoteAssetReserve)

	s.T().Log("check prices")
	priceInfo, err := testutilcli.QueryBaseAssetPrice(val.ClientCtx, common.Pair_ETH_NUSD, "add", "100")
	s.T().Logf("priceInfo: %+v", priceInfo)
	s.EqualValues(sdk.MustNewDecFromStr("599994.000059999400006000"), priceInfo.PriceInQuoteDenom)
	s.NoError(err)
}

func (s *IntegrationTestSuite) TestCmdEditPoolProposal() {
	val := s.network.Validators[0]
	clientCtx := val.ClientCtx.WithOutputFormat("json")
	govQueryClient := govtypes.NewQueryClient(clientCtx)

	// ----------------------------------------------------------------------
	s.T().Log("load example proposal json as bytes")
	// ----------------------------------------------------------------------
	proposal := &vpooltypes.EditPoolConfigProposal{
		Title:       "NIP-3: Edit config of the ueth:unusd vpool",
		Description: "enables higher max leverage on ueth:unusd",
		Pair:        common.Pair_ETH_NUSD.String(),
		Config: vpooltypes.VpoolConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.03"),
			MaxLeverage:            sdk.MustNewDecFromStr("25"),
		},
	}
	proposalFile := sdktestutil.WriteToNewTempFile(s.T(), string(val.ClientCtx.Codec.MustMarshalJSON(proposal)))
	contents, err := os.ReadFile(proposalFile.Name())
	s.Require().NoError(err)

	s.T().Log("Unmarshal json bytes into proposal object; check validity")
	proposal = &vpooltypes.EditPoolConfigProposal{}
	val.ClientCtx.Codec.MustUnmarshalJSON(contents, proposal)
	s.Require().NoError(proposal.ValidateBasic())

	// ----------------------------------------------------------------------
	s.T().Log("Submit proposal and unmarshal tx response")
	// ----------------------------------------------------------------------
	args := []string{
		proposalFile.Name(),
		fmt.Sprintf("--%s=1000unibi", govcli.FlagDeposit),
	}
	cmd := cli.CmdEditPoolConfigProposal()
	flags.AddTxFlagsToCmd(cmd)
	txResp, err := testutilcli.ExecTx(s.network, cmd, val.Address, args)
	s.Require().NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

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
	args = []string{
		/*proposal-id=*/ strconv.Itoa(int(proposalId)),
		/*deposit=*/ govDepositParams.DepositParams.MinDeposit.String(),
	}
	txResp, err = testutilcli.ExecTx(s.network, govcli.NewCmdDeposit(), val.Address, args)
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

	txResp, err = testutilcli.ExecTx(s.network, govcli.NewCmdVote(), val.Address, args)
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.Require().Eventuallyf(
		func() bool {
			proposalQueryResp, err := govQueryClient.Proposal(
				context.Background(), &govtypes.QueryProposalRequest{ProposalId: proposalId})
			s.Require().NoError(err)
			return govtypes.StatusPassed == proposalQueryResp.Proposal.Status
		},
		20*time.Second,
		2*time.Second,
		"proposal should pass after voting period")

	// ----------------------------------------------------------------------
	s.T().Log("verify that the newly proposed vpool config has been set")
	// ----------------------------------------------------------------------
	s.Require().NoError(s.network.WaitForNextBlock())

	vpoolsQueryResp := &vpooltypes.QueryAllPoolsResponse{}
	s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdGetVpools(), nil, vpoolsQueryResp))

	found := false
	for _, vpool := range vpoolsQueryResp.Pools {
		if vpool.Pair.String() == proposal.Pair {
			s.EqualValues(vpooltypes.Vpool{
				Pair:              common.MustNewAssetPair(proposal.Pair),
				BaseAssetReserve:  vpool.BaseAssetReserve,
				QuoteAssetReserve: vpool.QuoteAssetReserve,
				Config:            proposal.Config,
			}, vpool)
			found = true
		}
	}
	s.True(found, "pool does not exist")
}

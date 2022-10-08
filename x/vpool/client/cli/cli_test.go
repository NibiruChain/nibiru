package cli_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/simapp"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type VpoolCLISuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

func (s *VpoolCLISuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	app.SetPrefixes(app.AccountAddressPrefix)

	encodingConfig := simapp.MakeTestEncodingConfig()
	genesisState := simapp.NewTestGenesisStateFromDefault()
	vpoolGenesis := vpooltypes.DefaultGenesis()
	vpoolGenesis.Vpools = []vpooltypes.VPool{
		{
			Pair:                   common.Pair_ETH_NUSD,
			BaseAssetReserve:       sdk.NewDec(10_000_000),
			QuoteAssetReserve:      sdk.NewDec(60_000_000_000),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	}
	genesisState[vpooltypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(vpoolGenesis)

	s.cfg = testutilcli.BuildNetworkConfig(genesisState)

	s.network = testutilcli.NewNetwork(s.T(), s.cfg)
	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *VpoolCLISuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *VpoolCLISuite) TestGovAddVpool() {
	s.Require().Len(s.network.Validators, 1)
	val := s.network.Validators[0]
	clientCtx := val.ClientCtx.WithOutputFormat("json")
	govQueryClient := govtypes.NewQueryClient(clientCtx)

	// ----------------------------------------------------------------------
	s.T().Log("load example proposal json as bytes")
	// ----------------------------------------------------------------------
	proposal := &vpooltypes.CreatePoolProposal{
		Title:                  "Create ETH:USD pool",
		Description:            "Creates an ETH:USD pool",
		Pair:                   "ETH:USD",
		TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
		QuoteAssetReserve:      sdk.NewDec(1_000_000),
		BaseAssetReserve:       sdk.NewDec(1_000_000),
		FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.05"),
		MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.05"),
		MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
		MaxLeverage:            sdk.MustNewDecFromStr("15"),
	}
	proposalJSONString := val.ClientCtx.Codec.MustMarshalJSON(proposal)
	proposalJSON := sdktestutil.WriteToNewTempFile(
		s.T(), string(proposalJSONString),
	)
	contents, err := ioutil.ReadFile(proposalJSON.Name())
	s.Assert().NoError(err)

	s.T().Log("Unmarshal json bytes into proposal object; check validity")
	proposal = &vpooltypes.CreatePoolProposal{}
	val.ClientCtx.Codec.MustUnmarshalJSON(contents, proposal)
	s.Require().NoError(proposal.ValidateBasic())

	// ----------------------------------------------------------------------
	s.T().Log("Submit proposal and unmarshal tx response")
	// ----------------------------------------------------------------------
	args := []string{
		proposalJSON.Name(),
		fmt.Sprintf("--%s=1000unibi", govcli.FlagDeposit),
		fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
	}
	cmd := cli.CmdCreatePoolProposal()
	flags.AddTxFlagsToCmd(cmd)
	txResp, err := testutilcli.ExecTx(s.network, cmd, val.Address, args)
	s.Require().NoError(err)
	s.Assert().EqualValues(0, txResp.Code)

	// ----------------------------------------------------------------------
	s.T().Log(`Check that proposal was correctly submitted with gov client
$ nibid query gov proposal 1`)
	// ----------------------------------------------------------------------
	proposalsQueryResp, err := govQueryClient.Proposals(
		context.Background(), &govtypes.QueryProposalsRequest{},
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(proposalsQueryResp.Proposals)

	proposalId := proposalsQueryResp.Proposals[0].ProposalId
	s.Require().GreaterOrEqual(proposalId, uint64(1), "first proposal should have proposal ID of at least 1")
	s.Assert().Equalf(
		govtypes.StatusDepositPeriod,
		proposalsQueryResp.Proposals[0].Status,
		"proposal should be in deposit period as it hasn't passed min deposit")
	s.Assert().EqualValues(
		sdk.NewCoins(sdk.NewInt64Coin("unibi", 1000)),
		proposalsQueryResp.Proposals[0].TotalDeposit)

	// ----------------------------------------------------------------------
	s.T().Log(`Move proposal to vote status by meeting min deposit
$ nibid tx gov deposit [proposal-id] [deposit] [flags]`)
	// ----------------------------------------------------------------------
	govDepositParams, err := govQueryClient.Params(
		context.Background(), &govtypes.QueryParamsRequest{ParamsType: govtypes.ParamDeposit})
	s.Assert().NoError(err)
	args = []string{
		/*proposal-id=*/ fmt.Sprint(proposalId),
		/*deposit=*/ govDepositParams.DepositParams.MinDeposit.String(),
		fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
	}
	txResp, err = testutilcli.ExecTx(s.network, govcli.NewCmdDeposit(), val.Address, args)
	s.Require().NoError(err)
	s.Assert().EqualValues(0, txResp.Code)

	proposalQueryResponse, err := govQueryClient.Proposal(
		context.Background(), &govtypes.QueryProposalRequest{ProposalId: proposalId})
	s.Require().NoError(err)
	s.Assert().Equalf(
		govtypes.StatusVotingPeriod,
		proposalQueryResponse.Proposal.Status,
		"proposal should be in voting period since min deposit has been met")

	// ----------------------------------------------------------------------
	s.T().Log(`Vote on the proposal.
$ nibid tx gov vote [proposal-id] [option] [flags]
e.g. $ nibid tx gov vote 1 yes`)
	// ----------------------------------------------------------------------
	args = []string{
		/*proposal-id=*/ fmt.Sprint(proposalId),
		/*option=*/ "yes",
		fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
	}

	txResp, err = testutilcli.ExecTx(s.network, govcli.NewCmdVote(), val.Address, args)
	s.Assert().NoError(err)
	s.Assert().EqualValues(0, txResp.Code)

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
	s.Require().NoError(testutilcli.ExecQuery(s.network, cli.CmdGetVpools(), []string{}, vpoolsQueryResp))

	found := false
	for _, pool := range vpoolsQueryResp.Pools {
		if pool.Pair.String() == proposal.Pair {
			s.Assert().EqualValues(vpooltypes.VPool{
				Pair:                   common.MustNewAssetPair(proposal.Pair),
				BaseAssetReserve:       proposal.BaseAssetReserve,
				QuoteAssetReserve:      proposal.QuoteAssetReserve,
				TradeLimitRatio:        proposal.TradeLimitRatio,
				FluctuationLimitRatio:  proposal.FluctuationLimitRatio,
				MaxOracleSpreadRatio:   proposal.MaxOracleSpreadRatio,
				MaintenanceMarginRatio: proposal.MaintenanceMarginRatio,
				MaxLeverage:            proposal.MaxLeverage,
			}, pool)
			found = true
		}
	}
	s.Require().True(found, "pool does not exist")
}

func (s *VpoolCLISuite) TestGetPrices() {
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

func TestVpoolCLISuite(t *testing.T) {
	suite.Run(t, new(VpoolCLISuite))
}

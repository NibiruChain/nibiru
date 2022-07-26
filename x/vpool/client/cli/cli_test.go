package cli_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdktestutilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

func (s *IntegrationTestSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	app.SetPrefixes(app.AccountAddressPrefix)

	encCfg := app.MakeTestEncodingConfig()
	defaultAppGenesis := app.ModuleBasics.DefaultGenesis(encCfg.Marshaler)
	testAppGenesis := testapp.NewTestGenesisState(encCfg.Marshaler, defaultAppGenesis)
	s.cfg = testutilcli.BuildNetworkConfig(testAppGenesis)

	s.network = testutilcli.NewNetwork(s.T(), s.cfg)

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)

	res, err := testutilcli.QueryPrice(
		s.network.Validators[0].ClientCtx,
		common.PairGovStable.String(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(sdk.NewDec(10), res.Price.Price)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s IntegrationTestSuite) TestX_CmdAddOracleProposalAndVote() {
	s.Require().Len(s.network.Validators, 1)
	val := s.network.Validators[0]
	clientCtx := val.ClientCtx.WithOutputFormat("json")
	proposer, _, err := val.ClientCtx.Keyring.NewMnemonic(
		/* uid */ "proposer",
		/* language */ keyring.English,
		/* hdPath */ sdk.FullFundraiserPath,
		/* bip39Passphrase */ "",
		/* algo */ hd.Secp256k1,
	)
	s.Require().NoError(err)

	s.T().Log("Fill proposer wallet to pay gas for prosal")
	gasTokens := sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 100_000_000))
	oracle := sdk.AccAddress(proposer.GetPubKey().Address())
	_, err = testutilcli.FillWalletFromValidator(oracle, gasTokens, val, s.cfg.BondDenom)
	s.Require().NoError(err)

	s.T().Log("load example json as bytes")
	proposal := &vpooltypes.CreatePoolProposal{
		Title:                 "Create ETH:USD pool",
		Description:           "Creates an ETH:USD pool",
		Pair:                  "ETH:USD",
		TradeLimitRatio:       sdk.MustNewDecFromStr("0.10"),
		QuoteAssetReserve:     sdk.NewDec(1_000_000),
		BaseAssetReserve:      sdk.NewDec(1_000_000),
		FluctuationLimitRatio: sdk.MustNewDecFromStr("0.05"),
		MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.05"),
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

	s.T().Log("Submit proposal and unmarshal tx response")
	args := []string{
		proposalJSON.Name(),
		fmt.Sprintf("--%s=1000unibi", govcli.FlagDeposit),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
		fmt.Sprintf("--from=%s", val.Address.String()),
	}
	cmd := cli.CmdCreatePoolProposal()
	flags.AddTxFlagsToCmd(cmd)
	out, err := sdktestutilcli.ExecTestCLICmd(clientCtx, cmd, args)
	s.Require().NoError(err)
	s.Assert().NotContains(out.String(), "fail")
	var txRespProtoMessage proto.Message = &sdk.TxResponse{}
	s.Assert().NoError(
		clientCtx.Codec.UnmarshalJSON(out.Bytes(), txRespProtoMessage),
		out.String())
	txResp := txRespProtoMessage.(*sdk.TxResponse)
	err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
	s.Assert().NoError(err)
	s.Assert().EqualValues(0, txResp.Code, out.String())

	s.T().Log(`Check that proposal was correctly submitted with gov client
			$ nibid query gov proposal 1`)
	// the proposal tx won't be included until next block
	s.Assert().NoError(s.network.WaitForNextBlock())
	govQueryClient := govtypes.NewQueryClient(clientCtx)
	proposalsQueryResponse, err := govQueryClient.Proposals(
		context.Background(), &govtypes.QueryProposalsRequest{},
	)
	s.Require().NoError(err)
	s.Assert().NotEmpty(proposalsQueryResponse.Proposals)
	s.Assert().EqualValues(1, proposalsQueryResponse.Proposals[0].ProposalId,
		"first proposal should have proposal ID of 1")
	s.Assert().Equalf(
		govtypes.StatusDepositPeriod,
		proposalsQueryResponse.Proposals[0].Status,
		"proposal should be in deposit period as it hasn't passed min deposit")
	s.Assert().EqualValues(
		sdk.NewCoins(sdk.NewInt64Coin("unibi", 1_000)),
		proposalsQueryResponse.Proposals[0].TotalDeposit,
	)

	s.T().Log(`Move proposal to vote status by meeting min deposit
			$ nibid tx gov deposit [proposal-id] [deposit] [flags]`)
	govDepositParams, err := govQueryClient.Params(
		context.Background(), &govtypes.QueryParamsRequest{ParamsType: govtypes.ParamDeposit})
	s.Assert().NoError(err)
	args = []string{
		/*proposal-id=*/ "1",
		/*deposit=*/ govDepositParams.DepositParams.MinDeposit.String(),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
		fmt.Sprintf("--from=%s", val.Address.String()),
	}
	_, err = sdktestutilcli.ExecTestCLICmd(clientCtx, govcli.NewCmdDeposit(), args)
	s.Assert().NoError(err)

	s.Assert().NoError(s.network.WaitForNextBlock())
	govQueryClient = govtypes.NewQueryClient(clientCtx)
	proposalsQueryResponse, err = govQueryClient.Proposals(
		context.Background(), &govtypes.QueryProposalsRequest{})
	s.Require().NoError(err)
	s.Assert().Equalf(
		govtypes.StatusVotingPeriod,
		proposalsQueryResponse.Proposals[0].Status,
		"proposal should be in voting period since min deposit has been met")

	s.T().Log(`Vote on the proposal.
			$ nibid tx gov vote [proposal-id] [option] [flags]
			e.g. $ nibid tx gov vote 1 yes`)
	args = []string{
		/*proposal-id=*/ "1",
		/*option=*/ "yes",
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
		fmt.Sprintf("--from=%s", val.Address.String()),
	}
	_, err = sdktestutilcli.ExecTestCLICmd(clientCtx, govcli.NewCmdVote(), args)
	s.Assert().NoError(err)
	txResp = &sdk.TxResponse{}
	err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
	s.Assert().NoError(err)
	s.Assert().EqualValues(0, txResp.Code, out.String())

	s.Assert().NoError(s.network.WaitForNextBlock())
	s.Require().Eventuallyf(func() bool {
		proposalsQueryResponse, err = govQueryClient.Proposals(
			context.Background(), &govtypes.QueryProposalsRequest{})
		s.Require().NoError(err)
		return govtypes.StatusPassed == proposalsQueryResponse.Proposals[0].Status
	}, 20*time.Second, 2*time.Second,
		"proposal should pass after voting period")

	s.T().Log("verify that the new proposed pool exists")
	cmd = cli.CmdGetVpools()
	args = []string{}
	queryResp := &vpooltypes.QueryAllPoolsResponse{}
	s.Require().NoError(testutilcli.ExecQuery(s.network, cmd, args, queryResp))

	found := false
	for _, pool := range queryResp.Pools {
		if pool.Pair.String() == proposal.Pair {
			require.Equal(s.T(), pool, &vpooltypes.Pool{
				Pair:                  common.MustNewAssetPair(proposal.Pair),
				BaseAssetReserve:      proposal.BaseAssetReserve,
				QuoteAssetReserve:     proposal.QuoteAssetReserve,
				TradeLimitRatio:       proposal.TradeLimitRatio,
				FluctuationLimitRatio: proposal.FluctuationLimitRatio,
				MaxOracleSpreadRatio:  proposal.MaxOracleSpreadRatio,
			})
			found = true
		}
	}

	require.True(s.T(), found, "pool does not exist")
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

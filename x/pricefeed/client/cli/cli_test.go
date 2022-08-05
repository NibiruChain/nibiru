package cli_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdktestutilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/client/cli"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

const (
	genOracleAddress  = "nibi1zuxt7fvuxgj69mjxu3auca96zemqef5u2yemly"
	genOracleMnemonic = "kit soon capital dry sadness balance rival embark behind coast online struggle deer crush hospital during man monkey prison action custom wink utility arrive"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg        testutilcli.Config
	network    *testutilcli.Network
	oracleUIDs []string
	oracleMap  map[string]sdk.AccAddress
}

func (s *IntegrationTestSuite) setupOraclesForKeyring() {
	val := s.network.Validators[0]
	s.oracleUIDs = []string{"oracle", "wrongOracle"}

	for _, oracleUID := range s.oracleUIDs {
		info, _, err := val.ClientCtx.Keyring.NewMnemonic(
			/* uid */ oracleUID,
			/* language */ keyring.English,
			/* hdPath */ sdk.FullFundraiserPath,
			/* bip39Passphrase */ "",
			/* algo */ hd.Secp256k1)
		s.oracleMap[oracleUID] = sdk.AccAddress(info.GetPubKey().Address())
		s.Require().NoError(err)
	}

	info, err := val.ClientCtx.Keyring.NewAccount(
		/* uid */ "genOracle",
		/* mnemonic */ genOracleMnemonic,
		/* bip39Passphrase */ "",
		/* hdPath */ sdk.FullFundraiserPath,
		/* algo */ hd.Secp256k1,
	)
	s.oracleMap["genOracle"] = sdk.AccAddress(info.GetPubKey().Address())
	s.Require().NoError(err)

	_, _, err = val.ClientCtx.Keyring.NewMnemonic(
		/* uid */ "oracle",
		/* language */ keyring.English,
		/* hdPath */ sdk.FullFundraiserPath,
		/* bip39Passphrase */ "",
		/* algo */ hd.Secp256k1)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "public key already exists in keybase")
}

func (s *IntegrationTestSuite) SetupSuite() {
	/* 	Make test skip if -short is not used:
	All tests: `go test ./...`
	Unit tests only: `go test ./... -short`
	Integration tests only: `go test ./... -run Integration`
	https://stackoverflow.com/a/41407042/13305627 */
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	app.SetPrefixes(app.AccountAddressPrefix)

	s.cfg = testutilcli.BuildNetworkConfig(testapp.NewTestGenesisStateFromDefault())
	s.network = testutilcli.NewNetwork(s.T(), s.cfg)

	s.oracleMap = make(map[string]sdk.AccAddress)
	s.setupOraclesForKeyring()

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

func (s IntegrationTestSuite) TestGetPriceCmd() {
	testCases := []struct {
		name string
		args []string

		expectedPrice sdk.Dec
		expectErr     bool
		respType      proto.Message
	}{
		{
			name: "Get default price of collateral token",
			args: []string{
				common.PairCollStable.String(),
			},
			expectedPrice: sdk.NewDec(1),
			respType:      &pricefeedtypes.QueryPriceResponse{},
		},
		{
			name: "Get default price of governance token",
			args: []string{
				common.PairGovStable.String(),
			},
			expectedPrice: sdk.NewDec(10),
			respType:      &pricefeedtypes.QueryPriceResponse{},
		},
		{
			name: "Invalid pair returns an error",
			args: []string{
				"invalid:pair",
			},
			expectErr: true,
			respType:  &pricefeedtypes.QueryPriceResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdQueryPrice()
			queryResp := new(pricefeedtypes.QueryPriceResponse)
			err := testutilcli.ExecQuery(
				s.network, cmd,
				append(tc.args, fmt.Sprintf("--%s=json", tmcli.OutputFlag)),
				queryResp,
			)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Assert().Equal(tc.expectedPrice, queryResp.Price.Price)
				s.Assert().Equal(tc.args[0], queryResp.Price.PairID)
			}
		})
	}
}

func (s IntegrationTestSuite) TestGetRawPricesCmd() {
	testCases := []struct {
		name string
		args []string

		expectedPrice  sdk.Dec
		expectedExpiry time.Time
		expectErr      bool
		respType       proto.Message
	}{
		{
			name: "Get default price of collateral token",
			args: []string{
				common.PairCollStable.String(),
			},
			expectedPrice: sdk.NewDec(1),
			respType:      &pricefeedtypes.QueryRawPricesResponse{},
		},
		{
			name: "Get default price of governance token",
			args: []string{
				common.PairGovStable.String(),
			},
			expectedPrice: sdk.NewDec(10),
			respType:      &pricefeedtypes.QueryRawPricesResponse{},
		},
		{
			name: "Invalid pair returns an error",
			args: []string{
				"invalid:pair",
			},
			expectedPrice: sdk.NewDec(10),
			expectErr:     true,
			respType:      &pricefeedtypes.QueryRawPricesResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdQueryRawPrices()
			queryResp := new(pricefeedtypes.QueryRawPricesResponse)
			err := testutilcli.ExecQuery(
				s.network, cmd,
				append(tc.args, fmt.Sprintf("--%s=json", tmcli.OutputFlag)),
				queryResp,
			)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(len(queryResp.RawPrices), 1)
				s.Assert().Equal(tc.expectedPrice, queryResp.RawPrices[0].Price)
				s.Assert().Equal(genOracleAddress, queryResp.RawPrices[0].OracleAddress)
				// The initial prices are valid for one hour
				s.Assert().True(expireWithinHours(queryResp.RawPrices[0].GetExpiry(), 1))
			}
		})
	}
}
func expireWithinHours(t time.Time, hours time.Duration) bool {
	now := time.Now()
	return t.After(now) && t.Before(now.Add(hours*time.Hour))
}

func (s IntegrationTestSuite) TestPairsCmd() {
	val := s.network.Validators[0]

	oracle, _ := sdk.AccAddressFromBech32(genOracleAddress)
	testCases := []struct {
		name string

		expectedMarkets []pricefeedtypes.Market
		respType        proto.Message
	}{
		{
			name: "Get current pairs",
			expectedMarkets: []pricefeedtypes.Market{
				pricefeedtypes.NewMarket(common.PairGovStable, []sdk.AccAddress{oracle}, true),
				pricefeedtypes.NewMarket(common.PairCollStable, []sdk.AccAddress{oracle}, true),
				pricefeedtypes.NewMarket(common.PairBTCStable, []sdk.AccAddress{oracle}, true),
				pricefeedtypes.NewMarket(common.PairETHStable, []sdk.AccAddress{oracle}, true),
			},
			respType: &pricefeedtypes.QueryMarketsResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdQueryMarkets()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := sdktestutilcli.ExecTestCLICmd(clientCtx, cmd, nil)
			s.Require().NoError(err, out.String())
			s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

			txResp := tc.respType.(*pricefeedtypes.QueryMarketsResponse)
			err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
			s.Require().NoError(err)
			s.Assert().Equal(len(tc.expectedMarkets), len(txResp.Markets))

			for _, market := range txResp.Markets {
				s.Assert().Contains(tc.expectedMarkets, market)
			}
		})
	}
}
func (s IntegrationTestSuite) TestPricesCmd() {
	val := s.network.Validators[0]

	testCases := []struct {
		name string

		expectedPricePairs []pricefeedtypes.CurrentPriceResponse
		respType           proto.Message
	}{
		{
			name: "Get current prices",
			expectedPricePairs: []pricefeedtypes.CurrentPriceResponse{
				pricefeedtypes.NewCurrentPriceResponse(common.PairGovStable.String(), sdk.NewDec(10)),
				pricefeedtypes.NewCurrentPriceResponse(common.PairCollStable.String(), sdk.NewDec(1)),
			},
			respType: &pricefeedtypes.QueryPricesResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdQueryPrices()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := sdktestutilcli.ExecTestCLICmd(clientCtx, cmd, nil)
			s.Require().NoError(err, out.String())
			s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

			txResp := tc.respType.(*pricefeedtypes.QueryPricesResponse)
			err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
			s.Require().NoError(err)
			s.Assert().Equal(len(tc.expectedPricePairs), len(txResp.Prices))

			for _, priceResponse := range txResp.Prices {
				s.Assert().Contains(tc.expectedPricePairs, priceResponse, tc.expectedPricePairs)
			}
		})
	}
}

func (s IntegrationTestSuite) TestOraclesCmd() {
	val := s.network.Validators[0]

	testCases := []struct {
		name string
		args []string

		expectedOracles []string
		expectErr       bool
		respType        proto.Message
	}{
		{
			name: "Get the collateral oracles",
			args: []string{
				common.PairCollStable.String(),
			},
			expectedOracles: []string{genOracleAddress},
			respType:        &pricefeedtypes.QueryOraclesResponse{},
		},
		{
			name: "Get the governance oracles",
			args: []string{
				common.PairGovStable.String(),
			},
			expectedOracles: []string{genOracleAddress},
			respType:        &pricefeedtypes.QueryOraclesResponse{},
		},
		{
			name: "Invalid pair returns an error",
			args: []string{
				"invalid:pair",
			},
			expectErr:       false,
			expectedOracles: []string{},
			respType:        &pricefeedtypes.QueryOraclesResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdQueryOracles()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := sdktestutilcli.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err, out.String())
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*pricefeedtypes.QueryOraclesResponse)
				err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
				s.Require().NoError(err)
				s.Assert().Equal(tc.expectedOracles, txResp.Oracles)
			}
		})
	}
}
func (s IntegrationTestSuite) TestSetPriceCmd() {
	err := s.network.WaitForNextBlock()
	s.Require().NoError(err)

	val := s.network.Validators[0]

	gov, col := common.PairGovStable, common.PairCollStable
	now := time.Now()
	expireInOneHour := strconv.Itoa(int(now.Add(1 * time.Hour).Unix()))
	expiredTS := strconv.Itoa(int(now.Add(-1 * time.Hour).Unix()))

	gasFeeToken := sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 1_000_000))
	for _, oracleName := range []string{"genOracle", "wrongOracle"} {
		_, err = testutilcli.FillWalletFromValidator(
			/*addr=*/ s.oracleMap[oracleName],
			/*balanece=*/ gasFeeToken,
			/*Validator=*/ val,
			/*feesDenom=*/ s.cfg.BondDenom)
		s.Require().NoError(err)
	}

	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.DenomGov, sdk.NewInt(10))).String()),
	}
	testCases := []struct {
		name string
		args []string

		expectedPriceForPair map[string]sdk.Dec
		respType             proto.Message
		expectedCode         uint32
		fromOracle           string
	}{
		{
			name: "Set the price of the governance token",
			args: []string{
				gov.Token0, gov.Token1, "100", expireInOneHour,
			},
			expectedPriceForPair: map[string]sdk.Dec{
				gov.String(): sdk.NewDec(100)},
			respType:   &sdk.TxResponse{},
			fromOracle: "genOracle",
		},
		{
			name: "Set the price of the collateral token",
			args: []string{
				col.Token0, col.Token1, "0.85", expireInOneHour,
			},
			expectedPriceForPair: map[string]sdk.Dec{
				col.String(): sdk.MustNewDecFromStr("0.85")},
			respType:   &sdk.TxResponse{},
			fromOracle: "genOracle",
		},
		{
			name: "Use invalid oracle",
			args: []string{
				col.Token0, col.Token1, "0.5", expireInOneHour,
			},
			respType:     &sdk.TxResponse{},
			expectedCode: 6,
			fromOracle:   "wrongOracle",
		},
		{
			name: "Set invalid pair returns an error",
			args: []string{
				"invalid", "pair", "123", expireInOneHour,
			},
			expectedCode: 6,
			respType:     &sdk.TxResponse{},
			fromOracle:   "genOracle",
		},
		{
			name: "Set expired pair returns an error",
			args: []string{
				col.Token0, col.Token1, "100", expiredTS,
			},
			expectedCode: 3,
			respType:     &sdk.TxResponse{},
			fromOracle:   "genOracle",
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdPostPrice()
			clientCtx := val.ClientCtx

			commonArgs = append(commonArgs,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, s.oracleMap[tc.fromOracle]))
			out, err := sdktestutilcli.ExecTestCLICmd(clientCtx, cmd, append(tc.args, commonArgs...))
			s.Require().NoError(err)
			s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType))

			txResp := tc.respType.(*sdk.TxResponse)
			err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
			s.Require().NoError(err)
			s.Assert().Equal(tc.expectedCode, txResp.Code)

			for pairID, price := range tc.expectedPriceForPair {
				currentPrice, err := testutilcli.QueryRawPrice(clientCtx, pairID)
				s.Require().NoError(err)
				for _, rp := range currentPrice.RawPrices {
					found := false
					if rp.PairID == pairID {
						s.Assert().Equal(price, rp.Price)
						found = true
						break
					}
					s.Assert().True(found)
				}
			}
		})
	}
}

func (s IntegrationTestSuite) TestGetParamsCmd() {
	val := s.network.Validators[0]

	var pricefeedGenState pricefeedtypes.GenesisState
	s.cfg.Codec.MustUnmarshalJSON(
		s.cfg.GenesisState[pricefeedtypes.ModuleName],
		&pricefeedGenState,
	)

	testCases := []struct {
		name string

		respType       proto.Message
		expectedParams pricefeedtypes.Params
	}{
		{
			name:           "Get all params",
			respType:       &pricefeedtypes.QueryParamsResponse{},
			expectedParams: pricefeedGenState.Params,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdQueryParams()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := sdktestutilcli.ExecTestCLICmd(clientCtx, cmd, nil)
			s.Require().NoError(err, out.String())
			s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

			txResp := tc.respType.(*pricefeedtypes.QueryParamsResponse)
			err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
			s.Require().NoError(err)
			s.Assert().Equal(tc.expectedParams, txResp.Params)
		})
	}
}

func (s IntegrationTestSuite) TestX_CmdAddOracleProposalAndVote() {
	s.T().Log("Create oracle account")
	s.Require().Len(s.network.Validators, 1)
	val := s.network.Validators[0]
	clientCtx := val.ClientCtx.WithOutputFormat("json")
	oracleKeyringInfo, _, err := val.ClientCtx.Keyring.NewMnemonic(
		/* uid */ "delphi-oracle",
		/* language */ keyring.English,
		/* hdPath */ sdk.FullFundraiserPath,
		/* bip39Passphrase */ "",
		/* algo */ hd.Secp256k1,
	)
	s.Require().NoError(err)

	s.T().Log("Fill oracle wallet to pay gas on post price")
	gasTokens := sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 100_000_000))
	oracle := sdk.AccAddress(oracleKeyringInfo.GetPubKey().Address())
	_, err = testutilcli.FillWalletFromValidator(oracle, gasTokens, val, s.cfg.BondDenom)
	s.Require().NoError(err)

	s.T().Log("load example json as bytes")
	proposal := &pricefeedtypes.AddOracleProposal{
		Title:       "Cataclysm-004",
		Description: "Whitelists Delphi to post prices for OHM and BTC",
		Oracles:     []string{oracle.String()},
		Pairs:       []string{"ohm:usd", "btc:usd"},
	}
	proposalJSONString := fmt.Sprintf(`
		{
			"title": "%v",
			"description": "%v",
			"oracles": ["%v"],
			"pairs": ["%v", "%v"]
		}	
		`, proposal.Title, proposal.Description, proposal.Oracles[0],
		proposal.Pairs[0], proposal.Pairs[1],
	)
	proposalJSON := sdktestutil.WriteToNewTempFile(
		s.T(), proposalJSONString,
	)
	contents, err := ioutil.ReadFile(proposalJSON.Name())
	s.Assert().NoError(err)

	s.T().Log("Unmarshal json bytes into proposal object; check validity")
	encodingConfig := simappparams.MakeTestEncodingConfig()
	proposal = &pricefeedtypes.AddOracleProposal{}
	err = encodingConfig.Marshaler.UnmarshalJSON(contents, proposal)
	s.Assert().NoError(err)
	s.Require().NoError(proposal.Validate())

	s.T().Log("Submit proposal and unmarshal tx response")
	args := []string{
		proposalJSON.Name(),
		fmt.Sprintf("--%s=1000unibi", govcli.FlagDeposit),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
		fmt.Sprintf("--from=%s", val.Address.String()),
	}
	cmd := cli.CmdAddOracleProposal()
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

	s.T().Log("verify that the new proposed pairs have been added to the params")
	cmd = cli.CmdQueryParams()
	args = []string{}
	queryResp := &pricefeedtypes.QueryParamsResponse{}
	s.Require().NoError(testutilcli.ExecQuery(s.network, cmd, args, queryResp))
	proposalPairs := common.NewAssetPairs(proposal.Pairs...)
	expectedPairs := append(pricefeedtypes.DefaultPairs, proposalPairs...)
	s.Assert().EqualValues(expectedPairs, queryResp.Params.Pairs)

	s.T().Log("verify that the oracle was whitelisted with a query")
	cmd = cli.CmdQueryOracles()
	for _, pair := range proposalPairs {
		args = []string{pair.String()}
		queryResp := &pricefeedtypes.QueryOraclesResponse{}
		s.Assert().NoError(testutilcli.ExecQuery(s.network, cmd, args, queryResp))
		for _, proposalOracle := range proposal.Oracles {
			s.Assert().Contains(queryResp.Oracles, proposalOracle)
		}
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

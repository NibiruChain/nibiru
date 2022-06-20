package cli_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	simappparams "github.com/cosmos/ibc-go/v3/testing/simapp/params"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"

	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/client/cli"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
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

// NewPricefeedGen returns an x/pricefeed GenesisState to specify the module parameters.
func NewPricefeedGen() *pftypes.GenesisState {
	oracle := sdk.MustAccAddressFromBech32(genOracleAddress)

	pairs := common.AssetPairs{
		{
			Token0: common.PairGovStable.Token0,
			Token1: common.PairGovStable.Token1,
		},
		{
			Token0: common.PairCollStable.Token0,
			Token1: common.PairCollStable.Token1,
		},
	}

	oracles := []sdk.AccAddress{oracle}
	return &pftypes.GenesisState{
		Params: pftypes.Params{
			Pairs: pairs.Strings(),
		},
		PostedPrices: []pftypes.PostedPrice{
			{
				PairID:        pairs[0].Name(),
				OracleAddress: oracle,
				Price:         sdk.NewDec(10),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
			{
				PairID:        pairs[1].Name(),
				OracleAddress: oracle,
				Price:         sdk.OneDec(),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
		},
		GenesisOracles: oracles,
	}
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

	s.cfg = testutilcli.DefaultConfig()

	app.SetPrefixes(app.AccountAddressPrefix)
	genesisState := app.ModuleBasics.DefaultGenesis(s.cfg.Codec)

	pricefeedGenJson := s.cfg.Codec.MustMarshalJSON(NewPricefeedGen())
	genesisState[pftypes.ModuleName] = pricefeedGenJson

	s.cfg.GenesisState = genesisState

	s.network = testutilcli.New(s.T(), s.cfg)

	s.oracleMap = make(map[string]sdk.AccAddress)
	s.setupOraclesForKeyring()

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s IntegrationTestSuite) TestGetPriceCmd() {
	val := s.network.Validators[0]

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
				common.PairCollStable.Name(),
			},
			expectedPrice: sdk.NewDec(1),
			respType:      &pftypes.QueryPriceResponse{},
		},
		{
			name: "Get default price of governance token",
			args: []string{
				common.PairGovStable.Name(),
			},
			expectedPrice: sdk.NewDec(10),
			respType:      &pftypes.QueryPriceResponse{},
		},
		{
			name: "Invalid pair returns an error",
			args: []string{
				"invalid:pair",
			},
			expectErr: true,
			respType:  &pftypes.QueryPriceResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdPrice()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*pftypes.QueryPriceResponse)
				err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
				s.Require().NoError(err)
				s.Assert().Equal(tc.expectedPrice, txResp.Price.Price)
				s.Assert().Equal(tc.args[0], txResp.Price.PairID)
			}
		})
	}
}

func (s IntegrationTestSuite) TestGetRawPricesCmd() {
	val := s.network.Validators[0]

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
				common.PairCollStable.Name(),
			},
			expectedPrice: sdk.NewDec(1),
			respType:      &pftypes.QueryRawPricesResponse{},
		},
		{
			name: "Get default price of governance token",
			args: []string{
				common.PairGovStable.Name(),
			},
			expectedPrice: sdk.NewDec(10),
			respType:      &pftypes.QueryRawPricesResponse{},
		},
		{
			name: "Invalid pair returns an error",
			args: []string{
				"invalid:pair",
			},
			expectedPrice: sdk.NewDec(10),
			expectErr:     true,
			respType:      &pftypes.QueryRawPricesResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdRawPrices()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*pftypes.QueryRawPricesResponse)
				err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
				s.Require().NoError(err)
				s.Require().Equal(len(txResp.RawPrices), 1)
				s.Assert().Equal(tc.expectedPrice, txResp.RawPrices[0].Price)
				s.Assert().Equal(genOracleAddress, txResp.RawPrices[0].OracleAddress)
				// The initial prices are valid for one hour
				s.Assert().True(expireWithinHours(txResp.RawPrices[0].GetExpiry(), 1))
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

	gov, col := common.PairGovStable, common.PairCollStable
	oracle, _ := sdk.AccAddressFromBech32(genOracleAddress)
	testCases := []struct {
		name string

		expectedPairs pftypes.PairResponses
		respType      proto.Message
	}{
		{
			name: "Get current pairs",
			expectedPairs: pftypes.PairResponses{
				pftypes.NewPairResponse(gov.Token1, gov.Token0, []sdk.AccAddress{oracle}, true),
				pftypes.NewPairResponse(col.Token1, col.Token0, []sdk.AccAddress{oracle}, true),
			},
			respType: &pftypes.QueryPairsResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdPairs()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, nil)
			s.Require().NoError(err, out.String())
			s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

			txResp := tc.respType.(*pftypes.QueryPairsResponse)
			err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
			s.Require().NoError(err)
			s.Assert().Equal(len(tc.expectedPairs), len(txResp.Pairs))

			for _, p := range txResp.Pairs {
				s.Assert().Contains(tc.expectedPairs, p)
			}
		})
	}
}
func (s IntegrationTestSuite) TestPricesCmd() {
	val := s.network.Validators[0]

	testCases := []struct {
		name string

		expectedPricePairs []pftypes.CurrentPriceResponse
		respType           proto.Message
	}{
		{
			name: "Get current prices",
			expectedPricePairs: []pftypes.CurrentPriceResponse{
				pftypes.NewCurrentPriceResponse(common.PairGovStable.Name(), sdk.NewDec(10)),
				pftypes.NewCurrentPriceResponse(common.PairCollStable.Name(), sdk.NewDec(1)),
			},
			respType: &pftypes.QueryPricesResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdPrices()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, nil)
			s.Require().NoError(err, out.String())
			s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

			txResp := tc.respType.(*pftypes.QueryPricesResponse)
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
				common.PairCollStable.Name(),
			},
			expectedOracles: []string{genOracleAddress},
			respType:        &pftypes.QueryOraclesResponse{},
		},
		{
			name: "Get the governance oracles",
			args: []string{
				common.PairGovStable.Name(),
			},
			expectedOracles: []string{genOracleAddress},
			respType:        &pftypes.QueryOraclesResponse{},
		},
		{
			name: "Invalid pair returns an error",
			args: []string{
				"invalid:pair",
			},
			expectErr:       false,
			expectedOracles: []string{},
			respType:        &pftypes.QueryOraclesResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdOracles()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err, out.String())
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*pftypes.QueryOraclesResponse)
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
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
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
				gov.Name(): sdk.NewDec(100)},
			respType:   &sdk.TxResponse{},
			fromOracle: "genOracle",
		},
		{
			name: "Set the price of the collateral token",
			args: []string{
				col.Token0, col.Token1, "0.85", expireInOneHour,
			},
			expectedPriceForPair: map[string]sdk.Dec{
				col.Name(): sdk.MustNewDecFromStr("0.85")},
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
			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, append(tc.args, commonArgs...))
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

	testCases := []struct {
		name string

		respType       proto.Message
		expectedParams pftypes.Params
	}{
		{
			name:           "Get all params",
			respType:       &pftypes.QueryParamsResponse{},
			expectedParams: NewPricefeedGen().Params,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdQueryParams()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, nil)
			s.Require().NoError(err, out.String())
			s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

			txResp := tc.respType.(*pftypes.QueryParamsResponse)
			err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
			s.Require().NoError(err)
			s.Assert().Equal(tc.expectedParams, txResp.Params)
		})
	}
}

func (s IntegrationTestSuite) TestCmdAddOracleProposalAndVote() {
	s.Run("proposal to whitelist an oracle", func() {
		s.T().Log("Create oracle account and fill wallet")

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

		s.T().Log("Fill oracle wallet so they can pay gas on post price")

		gasTokens := sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 100_000_000))
		oracle := sdk.AccAddress(oracleKeyringInfo.GetPubKey().Address())
		_, err = testutilcli.FillWalletFromValidator(oracle, gasTokens, val, s.cfg.BondDenom)
		s.Require().NoError(err)

		s.T().Log("load example json as bytes")

		proposal := pftypes.AddOracleProposal{
			Title:       "Cataclysm-004",
			Description: "Whitelists Delphi to post prices for OHM and BTC",
			// Oracle:      oracleKeyringInfo.GetAddress().String(),
			Oracle: sample.AccAddress().String(),
			Pairs:  []string{"ohm:usd", "btc:usd"},
		}
		proposalJSONString := fmt.Sprintf(`
			{
				"title": "%v",
				"description": "%v",
				"oracle": "%v",
				"pairs": ["%v", "%v"],
				"deposit": "1000unibi"
			}	
			`, proposal.Title, proposal.Description, proposal.Oracle, proposal.Pairs[0],
			proposal.Pairs[1],
		)
		proposalJSON := sdktestutil.WriteToNewTempFile(
			s.T(), proposalJSONString,
		)
		contents, err := ioutil.ReadFile(proposalJSON.Name())
		s.Assert().NoError(err)

		s.T().Log("Unmarshal json bytes into proposal object; check validity")

		encodingConfig := simappparams.MakeTestEncodingConfig()
		proposalWithDeposit := &pftypes.AddOracleProposalWithDeposit{}
		err = encodingConfig.Marshaler.UnmarshalJSON(contents, proposalWithDeposit)
		s.Assert().NoError(err)
		s.Require().NoError(proposal.Validate())

		s.T().Log("Submit proposal and unmarshal tx response")

		cmd := cli.CmdAddOracleProposal()
		args := []string{
			proposalJSON.Name(),
			fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
			fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
			fmt.Sprintf("--from=%s", val.Address.String()),
		}
		out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
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

		s.T().Log("Check that proposal was correctly submitted with gov client")

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

		s.T().Log("Move proposal to vote status by meeting min deposit")

		govDepositParams, err := govQueryClient.Params(
			context.Background(), &govtypes.QueryParamsRequest{ParamsType: govtypes.ParamDeposit})
		s.Assert().NoError(err)

		args = []string{
			/*id=*/ "1",
			/*deposit=*/ govDepositParams.DepositParams.MinDeposit.String(),
			fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
			fmt.Sprintf("--%s=test", flags.FlagKeyringBackend),
			fmt.Sprintf("--from=%s", val.Address.String()),
		}
		_, err = clitestutil.ExecTestCLICmd(clientCtx, govcli.NewCmdDeposit(), args)
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

		s.T().Log("Vote on the proposal")
	})
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

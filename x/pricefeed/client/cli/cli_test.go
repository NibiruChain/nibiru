package cli_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"
	abcitypes "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/client/cli"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
)

const (
	genOracleAddress  = "nibi1zuxt7fvuxgj69mjxu3auca96zemqef5u2yemly"
	genOracleMnemonic = "kit soon capital dry sadness balance rival embark behind coast online struggle deer crush hospital during man monkey prison action custom wink utility arrive"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg       testutilcli.Config
	network   *testutilcli.Network
	oracleMap map[string]sdk.AccAddress
}

func (s *IntegrationTestSuite) setupOraclesForKeyring() {
	for _, o := range []string{"oracle", "wrongOracle"} {
		s.oracleMap[o] = testutilcli.NewAccount(s.network, o)
	}
	info, err := s.network.Validators[0].ClientCtx.Keyring.NewAccount(
		/* uid */ "genOracle",
		/* mnemonic */ genOracleMnemonic,
		/* bip39Passphrase */ "",
		/* hdPath */ sdk.FullFundraiserPath,
		/* algo */ hd.Secp256k1,
	)
	s.Require().NoError(err)
	s.oracleMap["genOracle"] = sdk.AccAddress(info.GetPubKey().Address())
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

	// TODO(heisenberg): pull the pricefeed genesis initialization to this function
	s.cfg = testutilcli.BuildNetworkConfig(simapp.NewTestGenesisStateFromDefault())
	s.network = testutilcli.NewNetwork(s.T(), s.cfg)
	s.Require().NoError(s.network.WaitForNextBlock())

	s.oracleMap = make(map[string]sdk.AccAddress)
	s.setupOraclesForKeyring()
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
	}{
		{
			name: "Get price of USDC",
			args: []string{
				common.Pair_USDC_NUSD.String(),
			},
			expectedPrice: sdk.NewDec(1),
		},
		{
			name: "Get price of NIBI",
			args: []string{
				common.Pair_NIBI_NUSD.String(),
			},
			expectedPrice: sdk.NewDec(10),
		},
		{
			name: "Invalid pair returns an error",
			args: []string{
				"invalid:pair",
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			var queryResp pricefeedtypes.QueryPriceResponse
			err := testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdQueryPrice(), tc.args, &queryResp)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.EqualValues(tc.expectedPrice, queryResp.Price.Price)
				s.EqualValues(tc.args[0], queryResp.Price.PairID)
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
				common.Pair_USDC_NUSD.String(),
			},
			expectedPrice: sdk.NewDec(1),
			respType:      &pricefeedtypes.QueryRawPricesResponse{},
		},
		{
			name: "Get default price of governance token",
			args: []string{
				common.Pair_NIBI_NUSD.String(),
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
			var queryResp pricefeedtypes.QueryRawPricesResponse
			err := testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdQueryRawPrices(), tc.args, &queryResp)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(len(queryResp.RawPrices), 1)
				s.Equal(tc.expectedPrice, queryResp.RawPrices[0].Price)
				s.Equal(genOracleAddress, queryResp.RawPrices[0].OracleAddress)
				// The initial prices are valid for one hour
				s.True(expireWithinHours(queryResp.RawPrices[0].GetExpiry(), 1))
			}
		})
	}
}

func expireWithinHours(t time.Time, hours time.Duration) bool {
	now := time.Now()
	return t.After(now) && t.Before(now.Add(hours*time.Hour))
}

func (s IntegrationTestSuite) TestPairsCmd() {
	oracleAddr := sdk.MustAccAddressFromBech32(genOracleAddress)
	testCases := []struct {
		name string

		expectedMarkets []pricefeedtypes.Market
		respType        proto.Message
	}{
		{
			name: "Get current pairs",
			expectedMarkets: []pricefeedtypes.Market{
				pricefeedtypes.NewMarket(common.Pair_NIBI_NUSD, []sdk.AccAddress{oracleAddr}, true),
				pricefeedtypes.NewMarket(common.Pair_USDC_NUSD, []sdk.AccAddress{oracleAddr}, true),
				pricefeedtypes.NewMarket(common.Pair_BTC_NUSD, []sdk.AccAddress{oracleAddr}, true),
				pricefeedtypes.NewMarket(common.Pair_ETH_NUSD, []sdk.AccAddress{oracleAddr}, true),
			},
			respType: &pricefeedtypes.QueryMarketsResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			var queryResp pricefeedtypes.QueryMarketsResponse
			s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdQueryMarkets(), nil, &queryResp))

			s.Equal(len(tc.expectedMarkets), len(queryResp.Markets))
			for _, market := range queryResp.Markets {
				s.Contains(tc.expectedMarkets, market)
			}
		})
	}
}

func (s IntegrationTestSuite) TestCurrentPricesCmd() {
	testCases := []struct {
		name string

		expectedPrices []pricefeedtypes.CurrentPriceResponse
		respType       proto.Message
	}{
		{
			name: "Get current prices",
			expectedPrices: []pricefeedtypes.CurrentPriceResponse{
				{
					PairID: common.Pair_NIBI_NUSD.String(),
					Price:  sdk.NewDec(10),
					Twap:   sdk.NewDec(10),
				},
				{
					PairID: common.Pair_USDC_NUSD.String(),
					Price:  sdk.NewDec(1),
					Twap:   sdk.NewDec(1),
				},
			},
			respType: &pricefeedtypes.QueryPricesResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			var queryResp pricefeedtypes.QueryPricesResponse
			s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdQueryPrices(), nil, &queryResp))

			s.Equal(len(tc.expectedPrices), len(queryResp.Prices))
			for _, price := range queryResp.Prices {
				s.Contains(tc.expectedPrices, price)
			}
		})
	}
}

func (s IntegrationTestSuite) TestGetOraclesCmd() {
	testCases := []struct {
		name            string
		args            []string
		expectedOracles []string
	}{
		{
			name: "Get the USDC oracles",
			args: []string{
				common.Pair_USDC_NUSD.String(),
			},
			expectedOracles: []string{genOracleAddress},
		},
		{
			name: "Get the governance oracles",
			args: []string{
				common.Pair_NIBI_NUSD.String(),
			},
			expectedOracles: []string{genOracleAddress},
		},
		{
			name: "Invalid pair returns an error",
			args: []string{
				"invalid:pair",
			},
			expectedOracles: []string{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			var queryResp pricefeedtypes.QueryOraclesResponse
			s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdQueryOracles(), tc.args, &queryResp))
			s.Equal(tc.expectedOracles, queryResp.Oracles)
		})
	}
}

func queryBankBalance(ctx client.Context, s IntegrationTestSuite, account sdk.AccAddress) (finalBalance banktypes.QueryAllBalancesResponse) {
	resp, err := banktestutil.QueryBalancesExec(ctx, account)
	s.Require().NoError(err)
	s.Require().NoError(ctx.Codec.UnmarshalJSON(resp.Bytes(), &finalBalance))
	return
}

func (s IntegrationTestSuite) TestSetPriceCmd() {
	val := s.network.Validators[0]

	currentTime := time.Now()
	oneHourFuture := strconv.Itoa(int(currentTime.Add(1 * time.Hour).Unix()))
	oneHourPast := strconv.Itoa(int(currentTime.Add(-1 * time.Hour).Unix()))

	gasFeeToken := sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 1000))

	for _, oracleName := range []string{"genOracle", "wrongOracle"} {
		s.NoError(testutilcli.FillWalletFromValidator(
			/*addr=*/ s.oracleMap[oracleName],
			/*balance=*/ gasFeeToken,
			/*Validator=*/ val,
			/*feesDenom=*/ s.cfg.BondDenom),
		)
	}

	testCases := []struct {
		name string
		args []string

		expectedPriceForPair map[string]sdk.Dec
		expectedFeePaid      sdk.Int
		respType             proto.Message
		expectedCode         uint32
		from                 string
	}{
		{
			name: "Set the price of NIBI",
			args: []string{
				common.DenomNIBI, common.DenomNUSD, "100", oneHourFuture,
			},
			expectedPriceForPair: map[string]sdk.Dec{
				common.Pair_NIBI_NUSD.String(): sdk.NewDec(100),
			},
			expectedFeePaid: sdk.NewInt(0),
			respType:        &sdk.TxResponse{},
			from:            "genOracle",
		},
		{
			name: "Set the price of the USDC",
			args: []string{
				common.DenomUSDC, common.DenomNUSD, "0.85", oneHourFuture,
			},
			expectedPriceForPair: map[string]sdk.Dec{
				common.Pair_USDC_NUSD.String(): sdk.MustNewDecFromStr("0.85"),
			},
			expectedFeePaid: sdk.NewInt(0),
			respType:        &sdk.TxResponse{},
			from:            "genOracle",
		},
		{
			name: "Use invalid oracle",
			args: []string{
				common.DenomUSDC, common.DenomNUSD, "0.5", oneHourFuture,
			},
			expectedFeePaid: sdk.NewInt(10), // Pay fee since this oracle is not whitelisted
			respType:        &sdk.TxResponse{},
			expectedCode:    6,
			from:            "wrongOracle",
		},
		{
			name: "Set invalid pair returns an error",
			args: []string{
				"invalid", "pair", "123", oneHourFuture,
			},
			expectedFeePaid: sdk.NewInt(10), // Invalid pair means that oracle is not whitelisted for this, needs to pay fees
			expectedCode:    6,
			respType:        &sdk.TxResponse{},
			from:            "genOracle",
		},
		{
			name: "Set expired pair returns an error",
			args: []string{
				common.DenomUSDC, common.DenomNUSD, "100", oneHourPast,
			},
			expectedCode:    3,
			expectedFeePaid: sdk.NewInt(0),
			respType:        &sdk.TxResponse{},
			from:            "genOracle",
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			clientCtx := val.ClientCtx

			bankBalanceStart := queryBankBalance(clientCtx, s, s.oracleMap[tc.from])

			txResp, err := testutilcli.ExecTx(s.network, cli.CmdPostPrice(), s.oracleMap[tc.from], tc.args, testutilcli.WithTxCanFail(true))
			s.Require().NoError(err)
			s.Equal(tc.expectedCode, txResp.Code)

			bankBalanceEnd := queryBankBalance(clientCtx, s, s.oracleMap[tc.from])
			s.Require().EqualValues(
				tc.expectedFeePaid.Int64(),
				bankBalanceStart.Balances.AmountOf(common.DenomNIBI).
					Sub(bankBalanceEnd.Balances.AmountOf(common.DenomNIBI)).
					Int64(),
			)

			for pairID, price := range tc.expectedPriceForPair {
				currentPrice, err := testutilcli.QueryRawPrice(clientCtx, pairID)
				s.Require().NoError(err)
				found := false
				for _, rawPrice := range currentPrice.RawPrices {
					if rawPrice.OracleAddress == s.oracleMap[tc.from].String() {
						s.Equal(price, rawPrice.Price)
						found = true
						break
					}
				}
				s.True(found)
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

		expectedParams pricefeedtypes.Params
	}{
		{
			name:           "Get all params",
			expectedParams: pricefeedGenState.Params,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			var queryResp pricefeedtypes.QueryParamsResponse
			s.Require().NoError(testutilcli.ExecQuery(val.ClientCtx, cli.CmdQueryParams(), nil, &queryResp))
			s.Equal(tc.expectedParams, queryResp.Params)
		})
	}
}

func (s IntegrationTestSuite) TestX_AddOracleProposalAndVote() {
	s.T().Log("Create oracle account")
	val := s.network.Validators[0]
	clientCtx := val.ClientCtx.WithOutputFormat("json")

	s.T().Log("Fill oracle wallet to pay gas on post price")
	gasTokens := sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 100*common.Precision))
	oracle := testutilcli.NewAccount(s.network, "delphi-oracle")
	s.NoError(testutilcli.FillWalletFromValidator(oracle, gasTokens, val, s.cfg.BondDenom))

	// ----------------------------------------------------------------------
	s.T().Log("load example proposal json as bytes")
	// ----------------------------------------------------------------------
	proposal := &pricefeedtypes.AddOracleProposal{
		Title:       "Cataclysm-004",
		Description: "Whitelists Delphi to post prices for OHM and BTC",
		Oracles:     []string{oracle.String()},
		Pairs:       []string{"ohm:usd", "btc:usd"},
	}
	proposalFile := sdktestutil.WriteToNewTempFile(s.T(), string(clientCtx.Codec.MustMarshalJSON(proposal)))
	contents, err := os.ReadFile(proposalFile.Name())
	s.Require().NoError(err)

	// ----------------------------------------------------------------------
	s.T().Log("Unmarshal json bytes into proposal object; check validity")
	// ----------------------------------------------------------------------
	proposal = &pricefeedtypes.AddOracleProposal{}
	clientCtx.Codec.MustUnmarshalJSON(contents, proposal)
	s.Require().NoError(proposal.Validate())

	// ----------------------------------------------------------------------
	s.T().Log(`Submit proposal and unmarshal tx response
	$ nibid tx gov submit-proposal add-oracle [proposal-json] --deposit=[deposit] [flags]`)
	// ----------------------------------------------------------------------
	args := []string{
		proposalFile.Name(),
		fmt.Sprintf("--%s=1000unibi", govcli.FlagDeposit),
	}
	cmd := cli.CmdAddOracleProposal()
	flags.AddTxFlagsToCmd(cmd)
	txResp, err := testutilcli.ExecTx(s.network, cmd, val.Address, args)
	s.Require().NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	testutilcli.PassGovProposal(s.Suite, s.network)

	// ----------------------------------------------------------------------
	s.T().Log("verify that the new proposed pairs have been added to the params")
	// ----------------------------------------------------------------------
	var queryResp pricefeedtypes.QueryParamsResponse
	s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdQueryParams(), nil, &queryResp))
	proposalPairs := common.NewAssetPairs(proposal.Pairs...)
	expectedPairs := append(pricefeedtypes.DefaultPairs, proposalPairs...)
	s.EqualValues(expectedPairs, queryResp.Params.Pairs)

	// ----------------------------------------------------------------------
	s.T().Log("verify that the oracle was whitelisted")
	// ----------------------------------------------------------------------
	for _, pair := range proposalPairs {
		args = []string{pair.String()}
		var queryResp pricefeedtypes.QueryOraclesResponse
		s.NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdQueryOracles(), args, &queryResp))
		for _, proposedOracle := range proposal.Oracles {
			s.Contains(queryResp.Oracles, proposedOracle)
		}
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

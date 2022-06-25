package cli_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktestutilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

	encCfg := app.MakeTestEncodingConfig()
	defaultAppGenesis := app.ModuleBasics.DefaultGenesis(encCfg.Marshaler)
	testAppGenesis := testapp.NewTestGenesisState(encCfg.Marshaler, defaultAppGenesis)
	s.cfg = testutilcli.BuildNetworkConfig(testAppGenesis)

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
			cmd := cli.CmdPrice()
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
			cmd := cli.CmdRawPrices()
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

		expectedPairs pricefeedtypes.PairResponses
		respType      proto.Message
	}{
		{
			name: "Get current pairs",
			expectedPairs: pricefeedtypes.PairResponses{
				pricefeedtypes.NewPairResponse(common.PairGovStable, []sdk.AccAddress{oracle}, true),
				pricefeedtypes.NewPairResponse(common.PairCollStable, []sdk.AccAddress{oracle}, true),
				pricefeedtypes.NewPairResponse(common.PairBTCStable, []sdk.AccAddress{oracle}, true),
				pricefeedtypes.NewPairResponse(common.PairETHStable, []sdk.AccAddress{oracle}, true),
			},
			respType: &pricefeedtypes.QueryPairsResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdPairs()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := sdktestutilcli.ExecTestCLICmd(clientCtx, cmd, nil)
			s.Require().NoError(err, out.String())
			s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

			txResp := tc.respType.(*pricefeedtypes.QueryPairsResponse)
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
			cmd := cli.CmdPrices()
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
			cmd := cli.CmdOracles()
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

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

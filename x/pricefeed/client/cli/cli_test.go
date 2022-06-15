package cli_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/client/cli"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
)

const (
	oracleAddress  = "nibi1zuxt7fvuxgj69mjxu3auca96zemqef5u2yemly"
	oracleMnemonic = "kit soon capital dry sadness balance rival embark behind coast online struggle deer crush hospital during man monkey prison action custom wink utility arrive"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

// NewPricefeedGen returns an x/pricefeed GenesisState to specify the module parameters.
func NewPricefeedGen() *pftypes.GenesisState {
	oracle, _ := sdk.AccAddressFromBech32(oracleAddress)

	return &pftypes.GenesisState{
		Params: pftypes.Params{
			Pairs: []pftypes.Pair{
				{
					Token0:  common.GovStablePool.Token0,
					Token1:  common.GovStablePool.Token1,
					Oracles: []sdk.AccAddress{oracle}, Active: true,
				},
				{
					Token0:  common.CollStablePool.Token0,
					Token1:  common.CollStablePool.Token1,
					Oracles: []sdk.AccAddress{oracle}, Active: true,
				},
			},
		},
		PostedPrices: []pftypes.PostedPrice{
			{
				PairID:        common.GovStablePool.PairID(),
				OracleAddress: oracle,
				Price:         sdk.NewDec(10),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
			{
				PairID:        common.CollStablePool.PairID(),
				OracleAddress: oracle,
				Price:         sdk.OneDec(),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
		},
	}
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

	// modification to pay fee with test bond denom "stake"
	app.SetPrefixes(app.AccountAddressPrefix)
	genesisState := app.ModuleBasics.DefaultGenesis(s.cfg.Codec)

	pricefeedGenJson := s.cfg.Codec.MustMarshalJSON(NewPricefeedGen())
	genesisState[pftypes.ModuleName] = pricefeedGenJson

	s.cfg.GenesisState = genesisState

	s.network = testutilcli.New(s.T(), s.cfg)

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
				common.CollStablePool.PairID(),
			},
			expectedPrice: sdk.NewDec(1),
			respType:      &pftypes.QueryPriceResponse{},
		},
		{
			name: "Get default price of governance token",
			args: []string{
				common.GovStablePool.PairID(),
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
				common.CollStablePool.PairID(),
			},
			expectedPrice: sdk.NewDec(1),
			respType:      &pftypes.QueryRawPricesResponse{},
		},
		{
			name: "Get default price of governance token",
			args: []string{
				common.GovStablePool.PairID(),
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
				s.Assert().Equal(oracleAddress, txResp.RawPrices[0].OracleAddress)
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

	gov, col := common.GovStablePool, common.CollStablePool
	oracle, _ := sdk.AccAddressFromBech32(oracleAddress)
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
				pftypes.NewCurrentPriceResponse(common.GovStablePool.PairID(), sdk.NewDec(10)),
				pftypes.NewCurrentPriceResponse(common.CollStablePool.PairID(), sdk.NewDec(1)),
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

			for _, pp := range txResp.Prices {
				s.Assert().Contains(tc.expectedPricePairs, pp)
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
				common.CollStablePool.PairID(),
			},
			expectedOracles: []string{oracleAddress},
			respType:        &pftypes.QueryOraclesResponse{},
		},
		{
			name: "Get the governance oracles",
			args: []string{
				common.GovStablePool.PairID(),
			},
			expectedOracles: []string{oracleAddress},
			respType:        &pftypes.QueryOraclesResponse{},
		},
		{
			name: "Invalid pair returns an error",
			args: []string{
				"invalid:pair",
			},
			expectErr: true,
			respType:  &pftypes.QueryOraclesResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdOracles()
			clientCtx := val.ClientCtx.WithOutputFormat("json")

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
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
	val := s.network.Validators[0]

	gov, col := common.GovStablePool, common.CollStablePool
	now := time.Now()
	expireInOneHour, expiredTS := strconv.Itoa(int(now.Add(1*time.Hour).Unix())), strconv.Itoa(int(now.Add(-1*time.Hour).Unix()))
	_, err := val.ClientCtx.Keyring.NewAccount(
		/* uid */ "oracle",
		/* mnemonic */ oracleMnemonic,
		/* bip39Passphrase */ "",
		/* hdPath */ sdk.FullFundraiserPath,
		/* algo */ hd.Secp256k1,
	)
	s.Require().NoError(err)
	info, _, err := val.ClientCtx.Keyring.NewMnemonic("wrongOracle", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)
	wrongOracleAddress := sdk.AccAddress(info.GetPubKey().Address())
	oracle, _ := sdk.AccAddressFromBech32(oracleAddress)
	gasFeeToken := sdk.NewCoins(sdk.NewInt64Coin("stake", 100_000_000))
	_, err = testutilcli.FillWalletFromValidator(wrongOracleAddress, gasFeeToken, val, s.cfg.BondDenom)
	s.Require().NoError(err)
	_, err = testutilcli.FillWalletFromValidator(oracle, gasFeeToken, val, s.cfg.BondDenom)
	s.Require().NoError(err)
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
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
			expectedPriceForPair: map[string]sdk.Dec{gov.PairID(): sdk.NewDec(100)},
			respType:             &sdk.TxResponse{},
			fromOracle:           "oracle",
		},
		{
			name: "Set the price of the collateral token",
			args: []string{
				col.Token0, col.Token1, "0.5", expireInOneHour,
			},
			expectedPriceForPair: map[string]sdk.Dec{col.PairID(): sdk.NewDec(2)},
			respType:             &sdk.TxResponse{},
			fromOracle:           "oracle",
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
			expectedCode: 5,
			respType:     &sdk.TxResponse{},
			fromOracle:   "oracle",
		},
		{
			name: "Set expired pair returns an error",
			args: []string{
				col.Token0, col.Token1, "100", expiredTS,
			},
			expectedCode: 3,
			respType:     &sdk.TxResponse{},
			fromOracle:   "oracle",
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdPostPrice()
			clientCtx := val.ClientCtx

			commonArgs = append(commonArgs, fmt.Sprintf("--%s=%s", flags.FlagFrom, tc.fromOracle))
			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, append(tc.args, commonArgs...))
			s.Require().NoError(err, out.String())
			s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

			txResp := tc.respType.(*sdk.TxResponse)
			err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
			s.Require().NoError(err)
			s.Assert().Equal(tc.expectedCode, txResp.Code, out.String())

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

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

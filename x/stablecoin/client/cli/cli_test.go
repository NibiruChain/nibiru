package cli_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/stablecoin/client/cli"
	stabletypes "github.com/NibiruChain/nibiru/x/stablecoin/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
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

	encodingConfig := app.MakeEncodingConfigAndRegister()
	genesisState := genesis.NewTestGenesisState(encodingConfig)

	// x/stablecoin genesis state
	stableGen := stabletypes.DefaultGenesis()
	stableGen.Params.IsCollateralRatioValid = true
	stableGen.ModuleAccountBalance = sdk.NewCoin(denoms.USDC, sdk.NewInt(10000*common.TO_MICRO))
	genesisState[stabletypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(stableGen)

	oracleGenesis := oracletypes.DefaultGenesisState()
	oracleGenesis.ExchangeRates = []oracletypes.ExchangeRateTuple{
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: sdk.NewDec(10)},
		{Pair: asset.Registry.Pair(denoms.USDC, denoms.NUSD), ExchangeRate: sdk.OneDec()},
	}
	oracleGenesis.Params.VotePeriod = 1_000

	genesisState[oracletypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(oracleGenesis)

	homeDir := s.T().TempDir()
	s.cfg = testutilcli.BuildNetworkConfig(genesisState)

	network, err := testutilcli.New(s.T(), homeDir, s.cfg)
	s.Require().NoError(err)

	s.network = network
	_, err = s.network.WaitForHeight(1)
	s.NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s IntegrationTestSuite) TestMintStableCmd() {
	val := s.network.Validators[0]
	minter := testutilcli.NewAccount(s.network, "minter2")

	s.NoError(testutilcli.FillWalletFromValidator(
		minter,
		sdk.NewCoins(
			sdk.NewInt64Coin(denoms.NIBI, 100*common.TO_MICRO),
			sdk.NewInt64Coin(denoms.USDC, 100*common.TO_MICRO),
		),
		val,
		s.cfg.BondDenom,
	))

	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(denoms.NIBI, sdk.NewInt(10))).String()),
	}
	s.Require().NoError(s.network.WaitForNextBlock())

	testCases := []struct {
		name string
		args []string

		expectedStable sdkmath.Int
		expectErr      bool
		respType       proto.Message
		expectedCode   uint32
	}{
		{
			name: "Mint correct amount",
			args: append([]string{
				"1000000unusd",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, "minter2"),
			}, commonArgs...),
			expectedStable: sdk.NewInt(1 * common.TO_MICRO),
			expectErr:      false,
			respType:       &sdk.TxResponse{},
			expectedCode:   0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.MintStableCmd()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			s.Require().NoError(s.network.WaitForNextBlock())
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.NoError(err, out.String())
				s.NoError(
					clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				tx, err := testutilcli.QueryTx(val.ClientCtx, txResp.TxHash)
				s.NoError(err)

				s.Require().Equal(tc.expectedCode, tx.Code, out.String())

				resp, err := clitestutil.QueryBalancesExec(clientCtx, minter)
				s.NoError(err)

				var balRes banktypes.QueryAllBalancesResponse
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.NoError(err)

				s.Require().Equal(
					balRes.Balances.AmountOf(denoms.NUSD), tc.expectedStable)
			}
		})
	}
}

func (s IntegrationTestSuite) TestBurnStableCmd() {
	val := s.network.Validators[0]
	burner := testutilcli.NewAccount(s.network, "burn")
	s.NoError(testutilcli.FillWalletFromValidator(
		burner,
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 20_000),
			sdk.NewInt64Coin(denoms.NUSD, 50*common.TO_MICRO),
		),
		val,
		s.cfg.BondDenom,
	))
	s.NoError(s.network.WaitForNextBlock())

	defaultBondCoinsString := sdk.NewCoins(sdk.NewCoin(denoms.NIBI, sdk.NewInt(10))).String()
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, defaultBondCoinsString),
	}

	testCases := []struct {
		name string
		args []string

		expectedStable   sdkmath.Int
		expectedColl     sdkmath.Int
		expectedGov      sdkmath.Int
		expectedTreasury sdk.Coins
		expectedEf       sdk.Coins
		expectErr        bool
		respType         proto.Message
		expectedCode     uint32
	}{
		{
			name: "Burn at 100% collRatio",
			args: append([]string{
				"50000000unusd",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, "burn"),
			}, commonArgs...),
			expectedStable:   sdk.ZeroInt(),
			expectedColl:     sdk.NewInt(50*common.TO_MICRO - 100_000), // Collateral minus 0,02% fees
			expectedGov:      sdk.NewInt(19_990),
			expectedTreasury: sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 50_000)),
			expectedEf:       sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 50_000)),
			expectErr:        false,
			respType:         &sdk.TxResponse{},
			expectedCode:     0,
		},
		// {
		// 	name: "Burn at 90% collRatio",
		// 	args: append([]string{
		// 		"100000000unusd",
		// 		fmt.Sprintf("--%s=%s", flags.FlagFrom, "burn")}, commonArgs...),
		// 	expectedStable: sdk.ZeroInt(),
		// 	expectedColl:   sdk.NewInt(90 * common.TO_MICRO),
		// 	expectedGov:    sdk.NewInt(1 * common.TO_MICRO),
		// 	expectErr:      false,
		// 	respType:       &sdk.TxResponse{},
		// 	expectedCode:   0,
		// },
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.BurnStableCmd()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			s.Require().NoError(s.network.WaitForNextBlock())

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.NoError(err, out.String())
				s.NoError(
					clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType),
					out.String(),
				)

				txResp := tc.respType.(*sdk.TxResponse)
				tx, err := testutilcli.QueryTx(val.ClientCtx, txResp.TxHash)
				s.NoError(err)
				s.Require().Equal(tc.expectedCode, tx.Code, out.String())

				resp, err := clitestutil.QueryBalancesExec(clientCtx, burner)
				s.NoError(err)

				var balRes banktypes.QueryAllBalancesResponse
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.NoError(err)

				s.Require().Equal(
					tc.expectedColl, balRes.Balances.AmountOf(denoms.USDC))
				s.Require().Equal(
					tc.expectedGov, balRes.Balances.AmountOf(denoms.NIBI))
				s.Require().Equal(
					tc.expectedStable, balRes.Balances.AmountOf(denoms.NUSD))

				// Query treasury pool balance
				resp, err = clitestutil.QueryBalancesExec(
					clientCtx, types.NewModuleAddress(common.TreasuryPoolModuleAccount))
				s.NoError(err)
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.NoError(err)

				s.Require().Equal(
					tc.expectedTreasury, balRes.Balances)

				// Query ecosystem fund balance
				resp, err = clitestutil.QueryBalancesExec(
					clientCtx,
					types.NewModuleAddress(stabletypes.StableEFModuleAccount))
				s.NoError(err)
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.NoError(err)

				s.Require().Equal(
					tc.expectedEf, balRes.Balances)
			}
		})
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

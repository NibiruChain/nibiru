package cli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/stablecoin/client/cli"
	stabletypes "github.com/NibiruChain/nibiru/x/stablecoin/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
)

const (
	// oracleAddress = "nibi17ppzhnuv68felpv7p0ya5j2n0uvvngjuqtuq4l"
	oracleAddress = "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

// NewPricefeedGen returns an x/pricefeed GenesisState to specify the module parameters.
func NewPricefeedGen() *pftypes.GenesisState {
	oracle := sdk.MustAccAddressFromBech32(oracleAddress)

	pairs := common.AssetPairs{
		common.PairGovStable, common.PairCollStable,
	}
	return &pftypes.GenesisState{
		Params: pftypes.Params{Pairs: pairs},
		PostedPrices: []pftypes.PostedPrice{
			{
				PairID: common.PairGovStable.String(),
				Oracle: oracle.String(),
				Price:  sdk.NewDec(10),
				Expiry: time.Now().Add(1 * time.Hour),
			},
			{
				PairID: common.PairCollStable.String(),
				Oracle: oracle.String(),
				Price:  sdk.OneDec(),
				Expiry: time.Now().Add(1 * time.Hour),
			},
		},
		GenesisOracles: []string{oracle.String()},
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

	encodingConfig := app.MakeTestEncodingConfig()
	defaultAppGenesis := app.NewDefaultGenesisState(encodingConfig.Marshaler)
	s.cfg = testutilcli.BuildNetworkConfig(defaultAppGenesis)

	// modification to pay fee with test bond denom "stake"
	app.SetPrefixes(app.AccountAddressPrefix)
	genesisState := app.ModuleBasics.DefaultGenesis(s.cfg.Codec)
	stableGen := stabletypes.DefaultGenesis()

	// IsCollateralRatioValid behavior testted in x/stablecoin/abci_test.go
	stableGen.Params.IsCollateralRatioValid = true
	stableGen.ModuleAccountBalance = sdk.NewCoin(common.DenomColl, sdk.NewInt(10000000000))

	stableGenJson := s.cfg.Codec.MustMarshalJSON(stableGen)
	genesisState[stabletypes.ModuleName] = stableGenJson

	pricefeedGenJson := s.cfg.Codec.MustMarshalJSON(NewPricefeedGen())
	genesisState[pftypes.ModuleName] = pricefeedGenJson

	s.cfg.GenesisState = genesisState

	s.network = testutilcli.NewNetwork(s.T(), s.cfg)

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

/*
Create a new wallet and attempt to fill it with the required balance.
Tokens are sent by the validator, 'val'.
*/
func (s IntegrationTestSuite) fillWalletFromValidator(
	addr sdk.AccAddress, balance sdk.Coins, val *testutilcli.Validator,
) sdk.AccAddress {
	_, err := banktestutil.MsgSendExec(
		val.ClientCtx,
		val.Address,
		addr,
		balance,
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		testutilcli.DefaultFeeString(s.cfg.BondDenom),
	)
	s.Require().NoError(err)

	return addr
}

func (s IntegrationTestSuite) TestMintStableCmd() {
	val := s.network.Validators[0]

	info, _, err := val.ClientCtx.Keyring.NewMnemonic("minter2", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)
	minterAddr := sdk.AccAddress(info.GetPubKey().Address())
	s.fillWalletFromValidator(
		minterAddr,
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 20_000),
			sdk.NewInt64Coin(common.DenomGov, 100_000_000),
			sdk.NewInt64Coin(common.DenomColl, 100_000_000),
		),
		val)

	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}

	testCases := []struct {
		name string
		args []string

		expectedStable sdk.Int
		expectErr      bool
		respType       proto.Message
		expectedCode   uint32
	}{
		{
			name: "Mint correct amount",
			args: append([]string{
				"1000000unusd",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, "minter2")}, commonArgs...),
			expectedStable: sdk.NewInt(1000000),
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
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(
					clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				resp, err := banktestutil.QueryBalancesExec(clientCtx, minterAddr)
				s.Require().NoError(err)

				var balRes banktypes.QueryAllBalancesResponse
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.Require().NoError(err)

				s.Require().Equal(
					balRes.Balances.AmountOf(common.DenomStable), tc.expectedStable)
			}
		})
	}
}

func (s IntegrationTestSuite) TestBurnStableCmd() {
	val := s.network.Validators[0]

	info, _, err := val.ClientCtx.Keyring.NewMnemonic(
		"burn", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)
	minterAddr := sdk.AccAddress(info.GetPubKey().Address())
	s.fillWalletFromValidator(
		minterAddr,
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 20000),
			sdk.NewInt64Coin(common.DenomStable, 50_000_000),
		),
		val,
	)

	err = s.network.WaitForNextBlock()
	s.Require().NoError(err)

	defaultBondCoinsString := sdk.NewCoins(
		sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf(
			"--%s=%s", flags.FlagFees, defaultBondCoinsString),
	}

	testCases := []struct {
		name string
		args []string

		expectedStable   sdk.Int
		expectedColl     sdk.Int
		expectedGov      sdk.Int
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
				fmt.Sprintf("--%s=%s", flags.FlagFrom, "burn")}, commonArgs...),
			expectedStable:   sdk.ZeroInt(),
			expectedColl:     sdk.NewInt(50_000_000 - 100_000), // Collateral minus 0,02% fees
			expectedGov:      sdk.ZeroInt(),
			expectedTreasury: sdk.NewCoins(sdk.NewInt64Coin(common.DenomColl, 50_000)),
			expectedEf:       sdk.NewCoins(sdk.NewInt64Coin(common.DenomColl, 50_000)),
			expectErr:        false,
			respType:         &sdk.TxResponse{},
			expectedCode:     0,
		},
		// {
		// 	name: "Burn at 90% collRatio",
		// 	args: append([]string{
		// 		"100000000unusd",
		// 		fmt.Sprintf("--%s=%s", flags.FlagFrom, "burn")}, commonArgs...),
		// 	expectedStable: sdk.NewInt(0),
		// 	expectedColl:   sdk.NewInt(90_000_000),
		// 	expectedGov:    sdk.NewInt(1_000_000),
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
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(
					clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType),
					out.String(),
				)

				txResp := tc.respType.(*sdk.TxResponse)
				err = val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp)
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				resp, err := banktestutil.QueryBalancesExec(clientCtx, minterAddr)
				s.Require().NoError(err)

				var balRes banktypes.QueryAllBalancesResponse
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.Require().NoError(err)

				s.Require().Equal(
					tc.expectedColl, balRes.Balances.AmountOf(common.DenomColl))
				s.Require().Equal(
					tc.expectedGov, balRes.Balances.AmountOf(common.DenomGov))
				s.Require().Equal(
					tc.expectedStable, balRes.Balances.AmountOf(common.DenomStable))

				// Query treasury pool balance
				resp, err = banktestutil.QueryBalancesExec(
					clientCtx, types.NewModuleAddress(common.TreasuryPoolModuleAccount))
				s.Require().NoError(err)
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.Require().NoError(err)

				s.Require().Equal(
					tc.expectedTreasury, balRes.Balances)

				// Query ecosystem fund balance
				resp, err = banktestutil.QueryBalancesExec(
					clientCtx,
					types.NewModuleAddress(stabletypes.StableEFModuleAccount))
				s.Require().NoError(err)
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.Require().NoError(err)

				s.Require().Equal(
					tc.expectedEf, balRes.Balances)
			}
		})
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

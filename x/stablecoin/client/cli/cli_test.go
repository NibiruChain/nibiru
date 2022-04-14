package cli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/MatrixDao/matrix/app"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"

	cli "github.com/MatrixDao/matrix/x/stablecoin/client/cli"
	utils "github.com/MatrixDao/matrix/x/testutil"

	"github.com/MatrixDao/matrix/x/common"
	stabletypes "github.com/MatrixDao/matrix/x/stablecoin/types"

	pricefeedtypes "github.com/MatrixDao/matrix/x/pricefeed/types"
	"github.com/MatrixDao/matrix/x/testutil/network"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"

	sdk "github.com/cosmos/cosmos-sdk/types"

	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const (
	oracleAddress = "matrix17ppzhnuv68felpv7p0ya5j2n0uvvngjuqtuq4l"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

type MsgPostPrices []pricefeedtypes.MsgPostPrice

// NewPricefeedGen returns an x/pricefeed GenesisState to specify the module parameters.
func NewPricefeedGen() *pricefeedtypes.GenesisState {
	oracle, _ := sdk.AccAddressFromBech32(oracleAddress)

	return &pricefeedtypes.GenesisState{
		Params: pricefeedtypes.Params{
			Markets: []pricefeedtypes.Market{
				{MarketID: common.GovStablePool, BaseAsset: common.GovDenom,
					QuoteAsset: common.CollDenom, Oracles: []sdk.AccAddress{oracle},
					Active: true},
				{MarketID: common.CollStablePool, BaseAsset: common.CollDenom,
					QuoteAsset: common.StableDenom, Oracles: []sdk.AccAddress{oracle},
					Active: true},
			},
		},
		PostedPrices: []pricefeedtypes.PostedPrice{
			{
				MarketID:      common.GovStablePool,
				OracleAddress: oracle,
				Price:         sdk.NewDec(10),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
			{
				MarketID:      common.CollStablePool,
				OracleAddress: oracle,
				Price:         sdk.OneDec(),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
		},
	}
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	s.cfg = utils.DefaultConfig()

	// modification to pay fee with test bond denom "stake"
	app.SetPrefixes(app.AccountAddressPrefix)
	genesisState := app.ModuleBasics.DefaultGenesis(s.cfg.Codec)
	stableGen := stabletypes.DefaultGenesis()
	stableGen.ModuleAccountBalance = sdk.NewCoin(common.CollDenom, sdk.NewInt(10000000000))

	stableGenJson := s.cfg.Codec.MustMarshalJSON(stableGen)
	genesisState[stabletypes.ModuleName] = stableGenJson

	pricefeedGenJson := s.cfg.Codec.MustMarshalJSON(NewPricefeedGen())
	genesisState[pricefeedtypes.ModuleName] = pricefeedGenJson

	s.cfg.GenesisState = genesisState

	s.network = network.New(s.T(), s.cfg)

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
	addr sdk.AccAddress, balance sdk.Coins, val *network.Validator,
) sdk.AccAddress {
	_, err := banktestutil.MsgSendExec(
		val.ClientCtx,
		val.Address,
		addr,
		balance,
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		utils.DefaultFeeString(s.cfg),
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
			sdk.NewInt64Coin(s.cfg.BondDenom, 20000),
			sdk.NewInt64Coin(common.GovDenom, 100000000),
			sdk.NewInt64Coin(common.CollDenom, 100000000),
		),
		val)

	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		//fmt.Sprintf("--%s=%s", flags.FlagKeyringBackend, "test"),
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
				"1000000uusdm",
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
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				resp, err := banktestutil.QueryBalancesExec(clientCtx, minterAddr)
				s.Require().NoError(err)

				var balRes banktypes.QueryAllBalancesResponse
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.Require().NoError(err)

				s.Require().Equal(
					balRes.Balances.AmountOf(common.StableDenom), tc.expectedStable)

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
			sdk.NewInt64Coin(common.StableDenom, 100000000),
		),
		val,
	)

	defaultBondCoinsString := sdk.NewCoins(
		sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		//fmt.Sprintf("--%s=%s", flags.FlagKeyringBackend, "test"),
		fmt.Sprintf(
			"--%s=%s", flags.FlagFees, defaultBondCoinsString),
	}

	testCases := []struct {
		name string
		args []string

		expectedStable sdk.Int
		expectedColl   sdk.Int
		expectedGov    sdk.Int
		expectErr      bool
		respType       proto.Message
		expectedCode   uint32
	}{
		{
			name: "Burn at 100% collRatio",
			args: append([]string{
				"100000000uusdm",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, "burn")}, commonArgs...),
			expectedStable: sdk.NewInt(0),
			expectedColl:   sdk.NewInt(100_000_000),
			expectedGov:    sdk.NewInt(0),
			expectErr:      false,
			respType:       &sdk.TxResponse{},
			expectedCode:   0,
		},
		// {
		// 	name: "Burn at 90% collRatio",
		// 	args: append([]string{
		// 		"100000000uusdm",
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
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				resp, err := banktestutil.QueryBalancesExec(clientCtx, minterAddr)
				s.Require().NoError(err)

				var balRes banktypes.QueryAllBalancesResponse
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.Require().NoError(err)

				s.Require().Equal(
					balRes.Balances.AmountOf(common.CollDenom), tc.expectedColl)
				s.Require().Equal(
					balRes.Balances.AmountOf(common.GovDenom), tc.expectedGov)
				s.Require().Equal(
					balRes.Balances.AmountOf(common.StableDenom), tc.expectedStable)

			}
		})
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

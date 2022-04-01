package cli_test

import (
	"fmt"
	"testing"

	"github.com/MatrixDao/matrix/app"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"

	cli "github.com/MatrixDao/matrix/x/stablecoin/client/cli"
	utils "github.com/MatrixDao/matrix/x/testutil"

	stabletypes "github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	s.cfg = network.DefaultConfig()

	// modification to pay fee with test bond denom "stake"
	genesisState := app.ModuleBasics.DefaultGenesis(s.cfg.Codec)
	gammGen := stabletypes.DefaultGenesis()

	gammGenJson := s.cfg.Codec.MustMarshalJSON(gammGen)
	genesisState[stabletypes.ModuleName] = gammGenJson
	s.cfg.GenesisState = genesisState

	s.network = network.New(s.T(), s.cfg)

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s IntegrationTestSuite) TestMintCmd() {
	val := s.network.Validators[0]

	info, _, err := val.ClientCtx.Keyring.NewMnemonic("NewJoinPoolAddr", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)

	newAddr := sdk.AccAddress(info.GetPubKey().Address())

	_, err = banktestutil.MsgSendExec(
		val.ClientCtx,
		val.Address,
		newAddr,
		sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 20000), sdk.NewInt64Coin("usdm", 1000)), fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		utils.DefaultFeeString(s.cfg),
	)
	s.Require().NoError(err)

	testCases := []struct {
		name string
		args []string

		expectErr    bool
		respType     proto.Message
		expectedCode uint32
	}{
		{
			"Mint correct amount", // matrixd tx stablecoin mint 100usdm --from=validator --keyring-backend=test --chain-id=testing --yes
			[]string{
				"100usdm",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, newAddr),
				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
			},
			false, &sdk.TxResponse{}, 0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		fmt.Println("----------------------------------------------------------------------------")
		fmt.Println(tc.name)
		fmt.Println(tc.args)
		fmt.Println("----------------------------------------------------------------------------")

		s.Run(tc.name, func() {
			cmd := cli.NewMintCmd()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())
			}
		})
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

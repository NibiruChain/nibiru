package cli_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"

	"github.com/NibiruChain/nibiru/x/pricefeed/client/cli"
	utils "github.com/NibiruChain/nibiru/x/testutil"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"

	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"

	sdk "github.com/cosmos/cosmos-sdk/types"

	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
)

const (
	/*
		Generated locally for the test

		address: nibi1zuxt7fvuxgj69mjxu3auca96zemqef5u2yemly
		kit soon capital dry sadness balance rival embark behind coast online struggle deer crush hospital during man monkey prison action custom wink utility arrive
	*/
	oracleAddress  = "nibi1zuxt7fvuxgj69mjxu3auca96zemqef5u2yemly"
	oracleMnemonic = "kit soon capital dry sadness balance rival embark behind coast online struggle deer crush hospital during man monkey prison action custom wink utility arrive"

	// This is max int - 1
	farAwayTimestamp = 253402300798
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

	s.cfg = utils.DefaultConfig()

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
		utils.DefaultFeeString(s.cfg.BondDenom),
	)
	s.Require().NoError(err)

	return addr
}

func (s IntegrationTestSuite) TestMintStableCmd() {
	val := s.network.Validators[0]

	_, err := val.ClientCtx.Keyring.NewAccount(
		/* uid */ "oracle",
		/* mnemonic */ oracleMnemonic,
		/* bip39Passphrase */ "",
		/* hdPath */ sdk.FullFundraiserPath,
		/* algo */ hd.Secp256k1,
	)
	s.Require().NoError(err)

	oracleAddr, _ := sdk.AccAddressFromBech32(oracleAddress)
	gasFeeToken := sdk.NewCoins(sdk.NewInt64Coin("stake", 100_000_000))
	s.fillWalletFromValidator(oracleAddr, gasFeeToken, val)

	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}

	testCases := []struct {
		name string
		args []string

		expectErr    bool
		respType     proto.Message
		expectedCode uint32
	}{
		{
			name: "Post a price",
			args: append([]string{
				common.GovStablePool.Token0,
				common.GovStablePool.Token1,
				"30000",
				strconv.Itoa(farAwayTimestamp),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, "oracle"),
			}, commonArgs...),
			expectErr:    false,
			respType:     &sdk.TxResponse{},
			expectedCode: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdPostPrice()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(
					clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String(),
				)

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				reserveAssets, err := testutilcli.QueryPrice(val.ClientCtx, common.GovStablePool.Token0,
					common.GovStablePool.Token1)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(reserveAssets)
				s.Require().Equal(0, 1)
			}
		})
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

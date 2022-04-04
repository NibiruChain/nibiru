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

	pricefeedcli "github.com/MatrixDao/matrix/x/pricefeed/client/cli"
	pricefeedtypes "github.com/MatrixDao/matrix/x/pricefeed/types"
	"github.com/MatrixDao/matrix/x/testutil/network"
	"github.com/cosmos/cosmos-sdk/client/flags"
	keycli "github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

type MsgPostPrices []pricefeedtypes.MsgPostPrice

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	s.cfg = utils.DefaultConfig()

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

// Create a new wallet and try to fill the wallet with the require balance. Tokens are sent by the validator
func (s IntegrationTestSuite) createWallet(name string, balance sdk.Coins, val *network.Validator) sdk.AccAddress {
	info, _, err := val.ClientCtx.Keyring.NewMnemonic(name, keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)

	newAddr := sdk.AccAddress(info.GetPubKey().Address())
	_, err = banktestutil.MsgSendExec(
		val.ClientCtx,
		val.Address,
		newAddr,
		balance,
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		utils.DefaultFeeString(s.cfg),
	)
	s.Require().NoError(err)
	return newAddr
}

func (s IntegrationTestSuite) TestMintCmd() {
	val := s.network.Validators[0]

	minterAddr := s.createWallet("minter", sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 20000), sdk.NewInt64Coin("usdm", 1000)), val)
	s.createWallet("oracle", sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 20000)), val)

	testCases := []struct {
		name       string
		args       []string
		postPrices MsgPostPrices

		expectErr    bool
		respType     proto.Message
		expectedCode uint32
	}{
		{
			"Mint correct amount", // matrixd tx stablecoin mint 100usdm --from=validator --keyring-backend=test --chain-id=testing --yes
			[]string{
				"100usdm",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, minterAddr),
				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
			},
			MsgPostPrices{
				pricefeedtypes.MsgPostPrice{
					MarketID: common.CollDenom,
					Price:    sdk.MustNewDecFromStr("1.0"),
					Expiry:   time.Now().Add(1 * time.Hour),
				},
				pricefeedtypes.MsgPostPrice{
					MarketID: common.GovDenom,
					Price:    sdk.MustNewDecFromStr("10.0"),
					Expiry:   time.Now().Add(1 * time.Hour),
				},
			},
			false, &sdk.TxResponse{}, 0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		fmt.Println("----------------------------------------------------------------------------")

		keys_list_cmd := keycli.ListKeysCmd()
		outt, _ := clitestutil.ExecTestCLICmd(val.ClientCtx, keys_list_cmd, []string{})

		fmt.Println(outt)

		fmt.Println("----------------------------------------------------------------------------")

		postPricecmd := pricefeedcli.CmdPostPrice()
		for _, ppQuery := range tc.postPrices {
			ppQuery := ppQuery

			fmt.Println("Posting prices : ")
			fmt.Println([]string{ppQuery.MarketID, ppQuery.Price.String(), ppQuery.Expiry.String(), "--from=oracle"})
			fmt.Println("--------------------------------")
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, postPricecmd, []string{ppQuery.MarketID, ppQuery.Price.String(), ppQuery.Expiry.String(), "--from=val"})
			fmt.Println(out)
			s.Require().NoError(err)

			// post-price [market-id] [price] [expiry]
		}

		fmt.Println("Prices ::------------------------------------------------------------------------------------------------")
		out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, pricefeedcli.CmdPrices(), []string{})
		fmt.Println(out)
		s.Require().NoError(err)

		s.Run(tc.name, func() {
			cmd := cli.MintStableCmd()
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

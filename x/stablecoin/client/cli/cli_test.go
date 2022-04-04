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

func NewPricefeedGen() *pricefeedtypes.GenesisState {
	oracle, _ := sdk.AccAddressFromBech32(oracleAddress)

	return &pricefeedtypes.GenesisState{
		Params: pricefeedtypes.Params{
			Markets: []pricefeedtypes.Market{
				{MarketID: common.GovPricePool, BaseAsset: common.GovDenom,
					QuoteAsset: common.CollDenom, Oracles: []sdk.AccAddress{oracle}, Active: true},
				{MarketID: common.CollPricePool, BaseAsset: common.StableDenom,
					QuoteAsset: common.CollDenom, Oracles: []sdk.AccAddress{oracle}, Active: true},
			},
		},
		PostedPrices: []pricefeedtypes.PostedPrice{
			{
				MarketID:      common.GovPricePool,
				OracleAddress: oracle,
				Price:         sdk.MustNewDecFromStr("10.00"),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
			{
				MarketID:      common.CollPricePool,
				OracleAddress: oracle,
				Price:         sdk.MustNewDecFromStr("1"),
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
	gammGen := stabletypes.DefaultGenesis()

	gammGenJson := s.cfg.Codec.MustMarshalJSON(gammGen)
	pricefeedGenJson := s.cfg.Codec.MustMarshalJSON(NewPricefeedGen())

	genesisState[stabletypes.ModuleName] = gammGenJson
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

// Create a new wallet and try to fill the wallet with the require balance. Tokens are sent by the validator
func (s IntegrationTestSuite) fillWalletFromValidator(walletAddress string, balance sdk.Coins, val *network.Validator) sdk.AccAddress {
	newAddr := sdk.AccAddress(walletAddress)
	_, err := banktestutil.MsgSendExec(
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

	info, _, err := val.ClientCtx.Keyring.NewMnemonic("minter2", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)
	minterAddr := info.GetPubKey().Address().String()
	s.fillWalletFromValidator(minterAddr, sdk.NewCoins(sdk.NewInt64Coin(s.cfg.BondDenom, 20000), sdk.NewInt64Coin("uusdm", 1000000000)), val)

	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		//fmt.Sprintf("--%s=%s", flags.FlagKeyringBackend, "test"),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}
	fmt.Println(info.GetAddress().String())

	testCases := []struct {
		name string
		args []string

		expectErr    bool
		respType     proto.Message
		expectedCode uint32
	}{
		{
			name: "Mint correct amount",
			args: append([]string{
				"10000000uusdm",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, "minter2")}, commonArgs...),
			expectErr:    false,
			respType:     &sdk.TxResponse{},
			expectedCode: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.MintStableCmd()
			clientCtx := val.ClientCtx
			fmt.Println(tc.args)

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			fmt.Println(out)
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

package testutil

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	dexcli "github.com/NibiruChain/nibiru/x/dex/client/cli"
	"github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/network"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")
	s.network = network.New(s.T(), s.cfg)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s IntegrationTestSuite) TestCreatePoolCmd() {
	val := s.network.Validators[0]

	info, _, err := val.ClientCtx.Keyring.NewMnemonic(
		"NewCreatePoolAddr",
		keyring.English,
		sdk.FullFundraiserPath,
		"",
		hd.Secp256k1,
	)
	s.Require().NoError(err)
	poolCreatorAddr := sdk.AccAddress(info.GetPubKey().Address())

	_, err = banktestutil.MsgSendExec(
		val.ClientCtx,
		/*from=*/ val.Address,
		/*to=*/ poolCreatorAddr,
		/*amount=*/ sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 20000),
			sdk.NewInt64Coin(fmt.Sprintf("%stoken", val.Moniker), 20000),
			sdk.NewInt64Coin(common.GovDenom, 1e9), // for pool creation fee
		),
		/*extraArgs*/
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		testutil.DefaultFeeString(s.cfg),
	)
	s.Require().NoError(err)

	testCases := []struct {
		name           string
		tokenWeights   string
		initialDeposit string
		swapFee        string
		exitFee        string
		extraArgs      []string
		expectedErr    error
		respType       proto.Message
		expectedCode   uint32
	}{
		{
			name:           "create pool with insufficient funds",
			tokenWeights:   "1stake, 1node0token",
			initialDeposit: "1000000000stake,10000000000node0token",
			swapFee:        "0.003",
			exitFee:        "0.003",
			extraArgs:      []string{},
			respType:       &sdk.TxResponse{},
			expectedCode:   5, // bankKeeper code for insufficient funds
		},
		{
			name:           "create pool with invalid weights",
			tokenWeights:   "0stake, 1node0token",
			initialDeposit: "10000stake,10000node0token",
			swapFee:        "0.003",
			exitFee:        "0.003",
			extraArgs:      []string{},
			expectedErr:    types.ErrInvalidCreatePoolArgs,
		},
		{
			name:           "create pool with deposit not matching weights",
			tokenWeights:   "1stake, 1node0token",
			initialDeposit: "10000foo,10000node0token",
			swapFee:        "0.003",
			exitFee:        "0.003",
			extraArgs:      []string{},
			expectedErr:    types.ErrInvalidCreatePoolArgs,
		},
		{
			name:           "create pool with sufficient funds",
			tokenWeights:   "1stake, 1node0token",
			initialDeposit: "10000stake,10000node0token",
			swapFee:        "0.003",
			exitFee:        "0.003",
			extraArgs:      []string{},
			respType:       &sdk.TxResponse{},
			expectedCode:   0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			out, err := ExecMsgCreatePool(s.T(), val.ClientCtx, poolCreatorAddr, tc.tokenWeights, tc.initialDeposit, tc.swapFee, tc.exitFee, tc.extraArgs...)
			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())
			}
		})
	}
}

func (s IntegrationTestSuite) TestNewJoinPoolCmd() {
	val := s.network.Validators[0]

	// create a new pool
	_, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ "5stake,5node0token",
		/*initialDeposit=*/ "100stake,100node0token",
		/*swapFee=*/ "0.01",
		/*exitFee=*/ "0.01",
	)
	s.Require().NoError(err)

	// create a new user address
	info, _, err := val.ClientCtx.Keyring.NewMnemonic(
		"NewJoinPoolAddr",
		keyring.English,
		sdk.FullFundraiserPath,
		"",
		hd.Secp256k1,
	)
	s.Require().NoError(err)
	poolCreatorAddr := sdk.AccAddress(info.GetPubKey().Address())

	// fund the user
	_, err = banktestutil.MsgSendExec(
		val.ClientCtx,
		/*from=*/ val.Address,
		/*to=*/ poolCreatorAddr,
		/*amount=*/ sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 20000),
			sdk.NewInt64Coin(fmt.Sprintf("%stoken", val.Moniker), 20000),
		),
		/*extraArgs*/
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		testutil.DefaultFeeString(s.cfg),
	)
	s.Require().NoError(err)

	testCases := []struct {
		name         string
		args         []string
		expectErr    bool
		respType     proto.Message
		expectedCode uint32
	}{
		{
			name: "join pool with insufficient balance",
			args: []string{
				fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, 1),
				fmt.Sprintf("--%s=%s", dexcli.FlagTokensIn, "1000000000stake,10000000000node0token"),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, poolCreatorAddr),
				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
			},
			expectErr:    false,
			respType:     &sdk.TxResponse{},
			expectedCode: 5, // bankKeeper code for insufficient funds
		},
		{
			name: "join pool with sufficient balance",
			args: []string{ // join-pool --pool-id=1 --tokens-in=100stake,100node0token --from=newAddr
				fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, 1),
				fmt.Sprintf("--%s=%s", dexcli.FlagTokensIn, "100stake,100node0token"),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, poolCreatorAddr),
				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
			},
			expectErr:    false,
			respType:     &sdk.TxResponse{},
			expectedCode: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := dexcli.CmdJoinPool()
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
	cfg := testutil.DefaultConfig()
	suite.Run(t, &IntegrationTestSuite{cfg: cfg})
}

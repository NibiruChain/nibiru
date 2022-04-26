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
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"
	tmcli "github.com/tendermint/tendermint/libs/cli"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network

	testAccount sdk.AccAddress
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")
	s.network = network.New(s.T(), s.cfg)

	val := s.network.Validators[0]

	// create a new user address
	info, _, err := val.ClientCtx.Keyring.NewMnemonic(
		"NewAddr",
		keyring.English,
		sdk.FullFundraiserPath,
		"iron fossil rug jazz mosquito sand kangaroo noble motor jungle job silk naive assume poverty afford twist critic start solid actual fetch flat fix",
		hd.Secp256k1,
	)
	s.Require().NoError(err)
	s.testAccount = sdk.AccAddress(info.GetPubKey().Address())

	// fund the user
	_, err = banktestutil.MsgSendExec(
		val.ClientCtx,
		/*from=*/ val.Address,
		/*to=*/ s.testAccount,
		/*amount=*/ sdk.NewCoins(
			sdk.NewInt64Coin(common.StableDenom, 20000),
			sdk.NewInt64Coin(common.CollDenom, 20000),
			sdk.NewInt64Coin(common.GovDenom, 2e9), // for pool creation fee and more for tx fees
		),
		/*extraArgs*/
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		testutil.DefaultFeeString(s.cfg),
	)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s IntegrationTestSuite) TestACreatePoolCmd() {
	val := s.network.Validators[0]

	testCases := []struct {
		name              string
		tokenWeights      string
		initialDeposit    string
		swapFee           string
		exitFee           string
		extraArgs         []string
		expectedErr       error
		respType          proto.Message
		expectedCode      uint32
		queryexpectedPass bool
		queryexpectedErr  string
		queryArgs         []string
	}{
		{
			name:              "create pool with insufficient funds",
			tokenWeights:      fmt.Sprintf("1%s, 1%s", common.GovDenom, common.StableDenom),
			initialDeposit:    fmt.Sprintf("1000000000%s,10000000000%s", common.GovDenom, common.StableDenom),
			swapFee:           "0.003",
			exitFee:           "0.003",
			extraArgs:         []string{},
			respType:          &sdk.TxResponse{},
			expectedCode:      5, // bankKeeper code for insufficient funds
			queryexpectedPass: false,
			queryexpectedErr:  "no pool for this id",
			queryArgs:         []string{"1"},
		},
		{
			name:              "create pool with invalid weights",
			tokenWeights:      fmt.Sprintf("0%s, 1%s", common.GovDenom, common.StableDenom),
			initialDeposit:    fmt.Sprintf("10000%s,10000%s", common.GovDenom, common.StableDenom),
			swapFee:           "0.003",
			exitFee:           "0.003",
			extraArgs:         []string{},
			expectedErr:       types.ErrInvalidCreatePoolArgs,
			queryexpectedPass: false,
			queryexpectedErr:  "no pool for this id",
			queryArgs:         []string{"1"},
		},
		{
			name:              "create pool with deposit not matching weights",
			tokenWeights:      "1unibi, 1uust",
			initialDeposit:    "1000foo,10000uust",
			swapFee:           "0.003",
			exitFee:           "0.003",
			extraArgs:         []string{},
			expectedErr:       types.ErrInvalidCreatePoolArgs,
			queryexpectedPass: false,
			queryexpectedErr:  "no pool for this id",
			queryArgs:         []string{"1"},
		},
		{
			name:              "create pool with sufficient funds",
			tokenWeights:      "1unibi, 1uust",
			initialDeposit:    "100unibi,100uust",
			swapFee:           "0.01",
			exitFee:           "0.01",
			extraArgs:         []string{},
			respType:          &sdk.TxResponse{},
			expectedCode:      0,
			queryexpectedPass: true,
			queryArgs:         []string{"1"},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			out, err := ExecMsgCreatePool(s.T(), val.ClientCtx, s.testAccount, tc.tokenWeights, tc.initialDeposit, tc.swapFee, tc.exitFee, tc.extraArgs...)
			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				// Query balance
				cmd := dexcli.CmdTotalPoolLiquidity()
				out, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, tc.queryArgs)

				if !tc.queryexpectedPass {
					s.Require().Contains(out.String(), tc.queryexpectedErr)
				} else {
					resp := types.QueryTotalPoolLiquidityResponse{}
					s.Require().NoError(err, out.String())
					s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &resp), out.String())
				}
			}
		})
	}
}

func (s IntegrationTestSuite) TestBNewJoinPoolCmd() {
	val := s.network.Validators[0]

	// create a new pool
	_, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ "5unibi,5uust",
		/*initialDeposit=*/ "100unibi,100uust",
		/*swapFee=*/ "0.01",
		/*exitFee=*/ "0.01",
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
				fmt.Sprintf("--%s=%s", dexcli.FlagTokensIn, "1000000000unibi,10000000000uust"),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, s.testAccount),
				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))).String()),
			},
			expectErr:    false,
			respType:     &sdk.TxResponse{},
			expectedCode: 5, // bankKeeper code for insufficient funds
		},
		{
			name: "join pool with sufficient balance",
			args: []string{ // join-pool --pool-id=1 --tokens-in=100unibi,100uust --from=newAddr
				fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, 1),
				fmt.Sprintf("--%s=%s", dexcli.FlagTokensIn, "100unibi,100uust"),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, s.testAccount),
				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))).String()),
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

func (s IntegrationTestSuite) TestCNewExitPoolCmd() {
	val := s.network.Validators[0]

	testCases := []struct {
		name               string
		args               []string
		expectErr          bool
		respType           proto.Message
		expectedCode       uint32
		expectedunibi      sdk.Int
		expectedOtherToken sdk.Int
	}{
		{
			name: "exit pool from invalid pool",
			args: []string{
				fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, 2),
				fmt.Sprintf("--%s=%s", dexcli.FlagPoolSharesOut, "100nibiru/pool/1"),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, s.testAccount),

				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))).String()),
			},
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1, // dex.types.ErrNonExistingPool
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name: "exit pool for too many shares",
			args: []string{
				fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, 1),
				fmt.Sprintf("--%s=%s", dexcli.FlagPoolSharesOut, "1001000000000000000000nibiru/pool/1"),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, s.testAccount),

				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))).String()),
			},
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1,
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name: "exit pool for zero shares",
			args: []string{
				fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, 1),
				fmt.Sprintf("--%s=%s", dexcli.FlagPoolSharesOut, "0nibiru/pool/1"),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, s.testAccount),

				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))).String()),
			},
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1,
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name: "exit pool with sufficient balance",
			args: []string{
				fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, 1),
				fmt.Sprintf("--%s=%s", dexcli.FlagPoolSharesOut, "101000000000000000000nibiru/pool/1"),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, s.testAccount),

				// common args
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))).String()),
			},
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       0,
			expectedunibi:      sdk.NewInt(100 - 10 - 1), // Received unibi minus 10unibi tx fee minus 1 exit pool fee
			expectedOtherToken: sdk.NewInt(100 - 1),      // Received uust minus 1 exit pool fee
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			// Get original balance
			resp, err := banktestutil.QueryBalancesExec(val.ClientCtx, s.testAccount)
			s.Require().NoError(err)
			var originalBalRes banktypes.QueryAllBalancesResponse
			err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &originalBalRes)
			s.Require().NoError(err)

			cmd := dexcli.CmdExitPool()

			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, tc.args)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				// Ensure balance is ok
				resp, err := banktestutil.QueryBalancesExec(val.ClientCtx, s.testAccount)
				s.Require().NoError(err)

				var balRes banktypes.QueryAllBalancesResponse
				err = val.ClientCtx.Codec.UnmarshalJSON(resp.Bytes(), &balRes)
				s.Require().NoError(err)

				fmt.Println("Final balance:")
				fmt.Println(balRes)

				s.Require().Equal(
					balRes.Balances.AmountOf("uust").Sub(
						originalBalRes.Balances.AmountOf("uust")).Sub(
						tc.expectedOtherToken).Int64(),
					sdk.NewInt(0).Int64(),
				)

				s.Require().Equal(
					balRes.Balances.AmountOf("unibi").Sub(
						originalBalRes.Balances.AmountOf("unibi")).Sub(
						tc.expectedunibi).Int64(),
					sdk.NewInt(0).Int64(),
				)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestDGetCmdTotalLiquidity() {
	val := s.network.Validators[0]

	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			"query total liquidity", // nibid query dex total-liquidity
			[]string{
				fmt.Sprintf("--%s=%s", tmcli.OutputFlag, "json"),
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := dexcli.CmdTotalLiquidity()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				resp := types.QueryTotalLiquidityResponse{}
				s.Require().NoError(err, out.String())
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &resp), out.String())
			}
		})
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	cfg := testutil.DefaultConfig()
	cfg.UpdateStartingToken(
		sdk.NewCoins(
			sdk.NewInt64Coin(common.StableDenom, 20000),
			sdk.NewInt64Coin(common.CollDenom, 20000),
			sdk.NewInt64Coin(common.GovDenom, 2e9), // for pool creation fee and more for tx fees
		),
	)
	suite.Run(t, &IntegrationTestSuite{cfg: cfg})
}

package testutil

import (
	"fmt"
	"testing"

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

	"github.com/NibiruChain/nibiru/x/common"
	dexcli "github.com/NibiruChain/nibiru/x/dex/client/cli"
	"github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network

	testAccount sdk.AccAddress
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
	s.network = testutilcli.New(s.T(), s.cfg)

	// create a new user address
	s.testAccount = s.NewAccount("NewAddr")

	// fund the user
	s.FundAccount(
		s.testAccount,
		sdk.NewCoins(
			sdk.NewInt64Coin(common.StableDenom, 20000),
			sdk.NewInt64Coin(common.CollDenom, 20000),
			sdk.NewInt64Coin(common.GovDenom, 2e9), // for pool creation fee and more for tx fees
		),
	)
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
			queryexpectedErr:  "pool not found",
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
			queryexpectedErr:  "pool not found",
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
			queryexpectedErr:  "pool not found",
			queryArgs:         []string{"1"},
		},
		{
			name:              "create pool with sufficient funds",
			tokenWeights:      "1unibi,1uust",
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
		poolId       uint64
		tokensIn     string
		expectErr    bool
		respType     proto.Message
		expectedCode uint32
	}{
		{
			name:         "join pool with insufficient balance",
			poolId:       1,
			tokensIn:     "1000000000unibi,10000000000uust",
			expectErr:    false,
			respType:     &sdk.TxResponse{},
			expectedCode: 5, // bankKeeper code for insufficient funds
		},
		{
			name:         "join pool with sufficient balance",
			poolId:       1,
			tokensIn:     "100unibi,100uust",
			expectErr:    false,
			respType:     &sdk.TxResponse{},
			expectedCode: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			ctx := val.ClientCtx

			out, err := ExecMsgJoinPool(s.T(), ctx, tc.poolId, s.testAccount, tc.tokensIn)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(ctx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

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
		poolId             uint64
		poolSharesOut      string
		expectErr          bool
		respType           proto.Message
		expectedCode       uint32
		expectedunibi      sdk.Int
		expectedOtherToken sdk.Int
	}{
		{
			name:               "exit pool from invalid pool",
			poolId:             2,
			poolSharesOut:      "100nibiru/pool/1",
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1, // dex.types.ErrNonExistingPool
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name:               "exit pool for too many shares",
			poolId:             1,
			poolSharesOut:      "1001000000000000000000nibiru/pool/1",
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1,
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name:               "exit pool for zero shares",
			poolId:             1,
			poolSharesOut:      "0nibiru/pool/1",
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1,
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name:               "exit pool with sufficient balance",
			poolId:             1,
			poolSharesOut:      "101000000000000000000nibiru/pool/1",
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       0,
			expectedunibi:      sdk.NewInt(100 - 10 - 1), // Received unibi minus 10unibi tx fee minus 1 exit pool fee
			expectedOtherToken: sdk.NewInt(100 - 1),      // Received uust minus 1 exit pool fee
		},
	}

	for _, tc := range testCases {
		tc := tc
		ctx := val.ClientCtx

		s.Run(tc.name, func() {
			// Get original balance
			resp, err := banktestutil.QueryBalancesExec(ctx, s.testAccount)
			s.Require().NoError(err)
			var originalBalance banktypes.QueryAllBalancesResponse
			s.Require().NoError(ctx.Codec.UnmarshalJSON(resp.Bytes(), &originalBalance))

			out, err := ExecMsgExitPool(s.T(), ctx, tc.poolId, s.testAccount, tc.poolSharesOut)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(ctx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				// Ensure balance is ok
				resp, err := banktestutil.QueryBalancesExec(ctx, s.testAccount)
				s.Require().NoError(err)
				var finalBalance banktypes.QueryAllBalancesResponse
				s.Require().NoError(ctx.Codec.UnmarshalJSON(resp.Bytes(), &finalBalance))

				fmt.Println("Final balance:")
				fmt.Println(finalBalance)

				s.Require().Equal(
					originalBalance.Balances.AmountOf("uust").Add(tc.expectedOtherToken),
					finalBalance.Balances.AmountOf("uust"),
				)
				s.Require().Equal(
					originalBalance.Balances.AmountOf("unibi").Add(tc.expectedunibi),
					finalBalance.Balances.AmountOf("unibi"),
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

func (s *IntegrationTestSuite) TestESwapAssets() {
	val := s.network.Validators[0]

	testCases := []struct {
		name          string
		poolId        uint64
		tokenIn       string
		tokenOutDenom string
		respType      proto.Message
		expectedCode  uint32
		expectErr     bool
	}{
		{
			name:          "zero pool id",
			poolId:        0,
			tokenIn:       "50unibi",
			tokenOutDenom: "uust",
			expectErr:     true,
		},
		{
			name:          "invalid token in",
			poolId:        1,
			tokenIn:       "0unibi",
			tokenOutDenom: "uust",
			expectErr:     true,
		},
		{
			name:          "invalid token out denom",
			poolId:        1,
			tokenIn:       "50unibi",
			tokenOutDenom: "",
			expectErr:     true,
		},
		{
			name:          "pool not found",
			poolId:        1000000,
			tokenIn:       "50unibi",
			tokenOutDenom: "uust",
			respType:      &sdk.TxResponse{},
			expectedCode:  types.ErrPoolNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token in denom not found",
			poolId:        1,
			tokenIn:       "50foo",
			tokenOutDenom: "uust",
			respType:      &sdk.TxResponse{},
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token out denom not found",
			poolId:        1,
			tokenIn:       "50unibi",
			tokenOutDenom: "foo",
			respType:      &sdk.TxResponse{},
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "successful swap",
			poolId:        1,
			tokenIn:       "50unibi",
			tokenOutDenom: "uust",
			respType:      &sdk.TxResponse{},
			expectedCode:  0,
			expectErr:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		ctx := val.ClientCtx

		s.Run(tc.name, func() {
			out, err := ExecMsgSwapAssets(s.T(), ctx, tc.poolId, s.testAccount, tc.tokenIn, tc.tokenOutDenom)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(ctx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())
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

/***************************** Convenience Methods ****************************/

/*
Adds tokens from val[0] to a recipient address.

args:
  - recipient: the recipient address
  - tokens: the amount of tokens to transfer
*/
func (s *IntegrationTestSuite) FundAccount(recipient sdk.Address, tokens sdk.Coins) {
	val := s.network.Validators[0]

	// fund the user
	_, err := banktestutil.MsgSendExec(
		val.ClientCtx,
		/*from=*/ val.Address,
		/*to=*/ recipient,
		/*amount=*/ tokens,
		/*extraArgs*/
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		testutil.DefaultFeeString(s.cfg),
	)
	s.Require().NoError(err)
}

/*
Creates a new account and returns the address.

args:
  - uid: a unique identifier to ensure duplicate accounts are not created

ret:
  - addr: the address of the new account
*/
func (s *IntegrationTestSuite) NewAccount(uid string) (addr sdk.AccAddress) {
	val := s.network.Validators[0]

	// create a new user address
	info, _, err := val.ClientCtx.Keyring.NewMnemonic(
		uid,
		keyring.English,
		sdk.FullFundraiserPath,
		"iron fossil rug jazz mosquito sand kangaroo noble motor jungle job silk naive assume poverty afford twist critic start solid actual fetch flat fix",
		hd.Secp256k1,
	)
	s.Require().NoError(err)

	return sdk.AccAddress(info.GetPubKey().Address())
}

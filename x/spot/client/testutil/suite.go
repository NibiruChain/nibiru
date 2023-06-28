package testutil

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"

	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/spot/client/cli"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	homeDir string
	cfg     testutilcli.Config
	network *testutilcli.Network
}

func NewIntegrationTestSuite(homeDir string, cfg testutilcli.Config) *IntegrationTestSuite {
	return &IntegrationTestSuite{homeDir: homeDir, cfg: cfg}
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

	network, err := testutilcli.New(
		s.T(),
		s.homeDir,
		s.cfg,
	)
	s.Require().NoError(err)

	s.network = network
	_, err = s.network.WaitForHeight(1)
	s.NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestCreatePoolCmd_Errors() {
	s.T().SkipNow()
	val := s.network.Validators[0]

	tc := []struct {
		name             string
		tokenWeights     string
		initialDeposit   string
		expectedErr      error
		expectedCode     uint32
		queryexpectedErr string
		queryArgs        []string
	}{
		{
			name:             "create pool with insufficient funds",
			tokenWeights:     fmt.Sprintf("1%s, 1%s", "coin-1", "coin-2"),
			initialDeposit:   fmt.Sprintf("1000000000%s,10000000000%s", "coin-1", "coin-2"),
			expectedCode:     5, // bankKeeper code for insufficient funds
			queryexpectedErr: "pool not found",
		},
		{
			name:             "create pool with invalid weights",
			tokenWeights:     fmt.Sprintf("0%s, 1%s", "coin-1", "coin-2"),
			initialDeposit:   fmt.Sprintf("10000%s,10000%s", "coin-1", "coin-2"),
			expectedErr:      types.ErrInvalidCreatePoolArgs,
			queryexpectedErr: "pool not found",
		},
		{
			name:             "create pool with deposit not matching weights",
			tokenWeights:     fmt.Sprintf("1%s, 1%s", "coin-1", "coin-2"),
			initialDeposit:   "1000foo,10000uusdc",
			expectedErr:      types.ErrInvalidCreatePoolArgs,
			queryexpectedErr: "pool not found",
		},
	}

	for _, tc := range tc {
		tc := tc

		s.Run(tc.name, func() {
			out, err := ExecMsgCreatePool(
				s.T(),
				val.ClientCtx,
				val.Address,
				tc.tokenWeights,
				tc.initialDeposit,
				"0.003",
				"0.003",
				"balancer",
				"0",
			)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)

				txResp := sdk.TxResponse{}
				s.network.Validators[0].ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), &txResp)
				s.Require().NoError(s.network.WaitForNextBlock())

				resp, err := testutilcli.QueryTx(s.network.Validators[0].ClientCtx, txResp.TxHash)
				s.Require().NoError(err)
				s.Assert().Equal(tc.expectedCode, resp.Code, string(s.network.Validators[0].ClientCtx.Codec.MustMarshalJSON(resp)))
			}
		})
	}
}

func (s *IntegrationTestSuite) TestCreatePoolStableSwapCmd_Errors() {
	s.T().SkipNow()
	val := s.network.Validators[0]

	tc := []struct {
		name           string
		amplification  string
		tokenWeights   string
		initialDeposit string
		poolType       string
		expectedErr    error
		expectedCode   uint32
	}{
		{
			name:          "create a stableswap pool, amplification parameter below 1",
			amplification: "0",
			poolType:      "stableswap",
			expectedCode:  17,
			expectedErr:   types.ErrAmplificationTooLow,
		},
		{
			name:          "create a balancer pool, no need for other parameters",
			amplification: "0",
			poolType:      "balancer",
			expectedErr:   nil,
		},
		{
			name:          "create a stableswap pool, happy path",
			amplification: "10",
			poolType:      "stableswap",
			expectedErr:   nil,
		},
	}

	for _, tc := range tc {
		tc := tc

		s.Run(tc.name, func() {
			tokenWeights := fmt.Sprintf("1%s, 1%s", "coin-1", "coin-2")
			initialDeposit := fmt.Sprintf("1000000000%s,10000000000%s", "coin-1", "coin-2")
			out, err := ExecMsgCreatePool(s.T(), val.ClientCtx, val.Address, tokenWeights, initialDeposit, "0.003", "0.003", tc.poolType, tc.amplification)
			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(s.network.WaitForNextBlock())

				resp := &sdk.TxResponse{}
				val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
				resp, err = testutilcli.QueryTx(val.ClientCtx, resp.TxHash)
				s.Require().NoError(err)

				s.Require().Equal(tc.expectedCode, resp.Code, string(val.ClientCtx.Codec.MustMarshalJSON(resp)))
			}
		})
	}
}

func (s *IntegrationTestSuite) TestCreatePoolCmd() {
	s.T().SkipNow()
	val := s.network.Validators[0]

	tc := []struct {
		name           string
		tokenWeights   string
		initialDeposit string
		poolType       string
		amplification  string
	}{
		{
			name:           "happy path",
			tokenWeights:   "1unibi,1uusdc",
			initialDeposit: "100unibi,100uusdc",
			poolType:       "balancer",
			amplification:  "0",
		},
		{
			name:           "happy path - stable",
			tokenWeights:   "1unusd,1uusdc",
			initialDeposit: "100unusd,100uusdc",
			poolType:       "stableswap",
			amplification:  "4",
		},
	}

	for _, tc := range tc {
		tc := tc

		s.Run(tc.name, func() {
			out, err := ExecMsgCreatePool(s.T(), val.ClientCtx, val.Address, tc.tokenWeights, tc.initialDeposit, "0.003", "0.003", tc.poolType, tc.amplification)
			s.Require().NoError(err, out.String())
			s.Require().NoError(s.network.WaitForNextBlock())

			resp := &sdk.TxResponse{}
			val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
			resp, err = testutilcli.QueryTx(s.network.Validators[0].ClientCtx, resp.TxHash)
			s.Require().NoError(err)

			s.Require().Equal(uint32(0), resp.Code, out.String())

			// Query balance
			cmd := cli.CmdTotalPoolLiquidity()
			out, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, []string{"1"})

			queryResp := types.QueryTotalPoolLiquidityResponse{}
			s.Require().NoError(err, out.String())
			val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), &queryResp)
			s.Require().EqualValues(tc.initialDeposit, queryResp.Liquidity)
		})
	}
}

func (s *IntegrationTestSuite) TestNewJoinPoolCmd() {
	val := s.network.Validators[0]

	// create a new pool
	out, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ fmt.Sprintf("5%s,5%s", "coin-2", "coin-3"),
		/*tokenWeights=*/ fmt.Sprintf("100%s,100%s", "coin-2", "coin-3"),
		/*swapFee=*/ "0.01",
		/*exitFee=*/ "0.01",
		/*poolType=*/ "balancer",
		/*amplification=*/ "0",
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.WaitForNextBlock())

	resp := &sdk.TxResponse{}
	val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
	resp, err = testutilcli.QueryTx(s.network.Validators[0].ClientCtx, resp.TxHash)
	s.Require().NoError(err, out.String())

	poolID, err := ExtractPoolIDFromCreatePoolResponse(val.ClientCtx.Codec, resp)
	s.Require().NoError(err)

	testCases := []struct {
		name         string
		poolId       uint64
		tokensIn     string
		expectErr    bool
		expectedCode uint32
	}{
		{
			name:         "join pool with insufficient balance",
			poolId:       poolID,
			tokensIn:     fmt.Sprintf("1000000000%s,10000000000%s", "coin-2", "coin-3"),
			expectErr:    false,
			expectedCode: 5, // bankKeeper code for insufficient funds
		},
		{
			name:         "join pool with wrong tokens",
			poolId:       poolID,
			tokensIn:     fmt.Sprintf("1000000000%s,10000000000%s", "coin-1", "coin-3"),
			expectErr:    false,
			expectedCode: 13, // bankKeeper code for insufficient funds
		},
		{
			name:         "join pool with sufficient balance",
			poolId:       poolID,
			tokensIn:     fmt.Sprintf("100%s,100%s", "coin-2", "coin-3"),
			expectErr:    false,
			expectedCode: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			ctx := val.ClientCtx

			out, err := ExecMsgJoinPool(ctx, tc.poolId, val.Address, tc.tokensIn, "false")
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(s.network.WaitForNextBlock())
				resp := &sdk.TxResponse{}
				ctx.Codec.MustUnmarshalJSON(out.Bytes(), resp)

				resp, err = testutilcli.QueryTx(ctx, resp.TxHash)
				s.Require().NoError(err, out.String())
				s.Require().Equal(tc.expectedCode, resp.Code, out.String())
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewJoinStablePoolCmd() {
	s.T().SkipNow()
	val := s.network.Validators[0]

	// create a new pool
	out, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ fmt.Sprintf("5%s,5%s", "coin-1", "coin-3"),
		/*tokenWeights=*/ fmt.Sprintf("100%s,100%s", "coin-1", "coin-3"),
		/*swapFee=*/ "0.01",
		/*exitFee=*/ "0.01",
		/*poolType=*/ "stableswap",
		/*amplification=*/ "10",
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.WaitForNextBlock())

	resp := &sdk.TxResponse{}
	val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
	resp, err = testutilcli.QueryTx(s.network.Validators[0].ClientCtx, resp.TxHash)
	s.Require().NoError(err)

	poolID, err := ExtractPoolIDFromCreatePoolResponse(val.ClientCtx.Codec, resp)
	s.Require().NoError(err, out.String())

	testCases := []struct {
		name         string
		poolId       uint64
		tokensIn     string
		expectErr    bool
		expectedCode uint32
	}{
		{
			name:         "join pool with insufficient balance",
			poolId:       poolID,
			tokensIn:     fmt.Sprintf("1000000000%s,10000000000%s", "coin-1", "coin-3"),
			expectErr:    false,
			expectedCode: 5, // bankKeeper code for insufficient funds
		},
		{
			name:         "join pool with sufficient balance",
			poolId:       poolID,
			tokensIn:     fmt.Sprintf("100%s,50%s", "coin-1", "coin-3"),
			expectErr:    false,
			expectedCode: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			ctx := val.ClientCtx

			out, err := ExecMsgJoinPool(ctx, tc.poolId, val.Address, tc.tokensIn, "false")
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(s.network.WaitForNextBlock())
				resp := &sdk.TxResponse{}
				ctx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
				resp, err = testutilcli.QueryTx(ctx, resp.TxHash)
				s.Require().NoError(err, out.String())

				s.Require().Equal(tc.expectedCode, resp.Code, out.String())
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewExitPoolCmd() {
	s.T().Skip("this test looks like it has a bug https://github.com/NibiruChain/nibiru/issues/869")
	val := s.network.Validators[0]

	// create a new pool
	out, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ fmt.Sprintf("1%s,1%s", "coin-3", "coin-4"),
		/*tokenWeights=*/ fmt.Sprintf("100%s,100%s", "coin-3", "coin-4"),
		/*swapFee=*/ "0.01",
		/*exitFee=*/ "0.01",
		/*poolType=*/ "balancer",
		/*amplification=*/ "0",
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.WaitForNextBlock())

	resp := &sdk.TxResponse{}
	val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
	resp, err = testutilcli.QueryTx(s.network.Validators[0].ClientCtx, resp.TxHash)
	s.Require().NoError(err)

	poolID, err := ExtractPoolIDFromCreatePoolResponse(val.ClientCtx.Codec, resp)
	s.Require().NoError(err, out.String())

	testCases := []struct {
		name               string
		poolId             uint64
		poolSharesOut      string
		expectErr          bool
		respType           proto.Message
		expectedCode       uint32
		expectedunibi      sdkmath.Int
		expectedOtherToken sdkmath.Int
	}{
		{
			name:               "exit pool from invalid pool",
			poolId:             100,
			poolSharesOut:      "100nibiru/pool/100",
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1, // spot.types.ErrNonExistingPool
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name:               "exit pool for too many shares",
			poolId:             poolID,
			poolSharesOut:      fmt.Sprintf("1001000000000000000000nibiru/pool/%d", poolID),
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1,
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name:               "exit pool for zero shares",
			poolId:             poolID,
			poolSharesOut:      fmt.Sprintf("0nibiru/pool/%d", poolID),
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1,
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{ // Looks with a bug
			name:               "exit pool with sufficient balance",
			poolId:             poolID,
			poolSharesOut:      fmt.Sprintf("100000000000000000000nibiru/pool/%d", poolID),
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       0,
			expectedunibi:      sdk.NewInt(100 - 10 - 1), // Received unibi minus 10unibi tx fee minus 1 exit pool fee
			expectedOtherToken: sdk.NewInt(100 - 1),      // Received uusdc minus 1 exit pool fee
		},
	}

	for _, tc := range testCases {
		tc := tc
		ctx := val.ClientCtx

		s.Run(tc.name, func() {
			// Get original balance
			resp, err := clitestutil.QueryBalancesExec(ctx, val.Address)
			s.Require().NoError(err)
			var originalBalance banktypes.QueryAllBalancesResponse
			s.Require().NoError(ctx.Codec.UnmarshalJSON(resp.Bytes(), &originalBalance))

			out, err := ExecMsgExitPool(ctx, tc.poolId, val.Address, tc.poolSharesOut)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(ctx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				// Ensure balance is ok
				resp, err := clitestutil.QueryBalancesExec(ctx, val.Address)
				s.Require().NoError(err)
				var finalBalance banktypes.QueryAllBalancesResponse
				s.Require().NoError(ctx.Codec.UnmarshalJSON(resp.Bytes(), &finalBalance))

				s.Require().Equal(
					originalBalance.Balances.AmountOf("uusdc").Add(tc.expectedOtherToken),
					finalBalance.Balances.AmountOf("uusdc"),
				)
				s.Require().Equal(
					originalBalance.Balances.AmountOf("unibi").Add(tc.expectedunibi),
					finalBalance.Balances.AmountOf("unibi"),
				)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewExitStablePoolCmd() {
	s.T().Skip("this test looks like it has a bug https://github.com/NibiruChain/nibiru/issues/869")
	val := s.network.Validators[0]

	// create a new pool
	out, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ fmt.Sprintf("1%s,1%s", "coin-3", "coin-4"),
		/*tokenWeights=*/ fmt.Sprintf("100%s,100%s", "coin-3", "coin-4"),
		/*swapFee=*/ "0.01",
		/*exitFee=*/ "0.01",
		/*poolType=*/ "stableswap",
		/*amplification=*/ "4242",
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.WaitForNextBlock())

	resp := &sdk.TxResponse{}
	val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
	resp, err = testutilcli.QueryTx(s.network.Validators[0].ClientCtx, resp.TxHash)
	s.Require().NoError(err)

	poolID, err := ExtractPoolIDFromCreatePoolResponse(val.ClientCtx.Codec, resp)
	s.Require().NoError(err, out.String())

	testCases := []struct {
		name               string
		poolId             uint64
		poolSharesOut      string
		expectErr          bool
		respType           proto.Message
		expectedCode       uint32
		expectedunibi      sdkmath.Int
		expectedOtherToken sdkmath.Int
	}{
		{
			name:               "exit pool from invalid pool",
			poolId:             100,
			poolSharesOut:      "100nibiru/pool/100",
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1, // spot.types.ErrNonExistingPool
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name:               "exit pool for too many shares",
			poolId:             poolID,
			poolSharesOut:      fmt.Sprintf("1001000000000000000000nibiru/pool/%d", poolID),
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1,
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{
			name:               "exit pool for zero shares",
			poolId:             poolID,
			poolSharesOut:      fmt.Sprintf("0nibiru/pool/%d", poolID),
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       1,
			expectedunibi:      sdk.NewInt(-10),
			expectedOtherToken: sdk.NewInt(0),
		},
		{ // Looks with a bug
			name:               "exit pool with sufficient balance",
			poolId:             poolID,
			poolSharesOut:      fmt.Sprintf("100000000000000000000nibiru/pool/%d", poolID),
			expectErr:          false,
			respType:           &sdk.TxResponse{},
			expectedCode:       0,
			expectedunibi:      sdk.NewInt(100 - 10 - 1), // Received unibi minus 10unibi tx fee minus 1 exit pool fee
			expectedOtherToken: sdk.NewInt(100 - 1),      // Received uusdc minus 1 exit pool fee
		},
	}

	for _, tc := range testCases {
		tc := tc
		ctx := val.ClientCtx

		s.Run(tc.name, func() {
			// Get original balance
			resp, err := clitestutil.QueryBalancesExec(ctx, val.Address)
			s.Require().NoError(err)
			var originalBalance banktypes.QueryAllBalancesResponse
			s.Require().NoError(ctx.Codec.UnmarshalJSON(resp.Bytes(), &originalBalance))

			out, err := ExecMsgExitPool(ctx, tc.poolId, val.Address, tc.poolSharesOut)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(ctx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())

				// Ensure balance is ok
				resp, err := clitestutil.QueryBalancesExec(ctx, val.Address)
				s.Require().NoError(err)
				var finalBalance banktypes.QueryAllBalancesResponse
				s.Require().NoError(ctx.Codec.UnmarshalJSON(resp.Bytes(), &finalBalance))

				s.Require().Equal(
					originalBalance.Balances.AmountOf("uusdc").Add(tc.expectedOtherToken),
					finalBalance.Balances.AmountOf("uusdc"),
				)
				s.Require().Equal(
					originalBalance.Balances.AmountOf("unibi").Add(tc.expectedunibi),
					finalBalance.Balances.AmountOf("unibi"),
				)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdTotalLiquidity() {
	val := s.network.Validators[0]

	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			"query total liquidity", // nibid query spot total-liquidity
			[]string{
				fmt.Sprintf("--%s=%s", tmcli.OutputFlag, "json"),
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.CmdTotalLiquidity()
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

func (s *IntegrationTestSuite) TestSwapAssets() {
	s.T().SkipNow()
	val := s.network.Validators[0]

	// create a new pool
	out, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ fmt.Sprintf("1%s,1%s", "coin-4", "coin-5"),
		/*tokenWeights=*/ fmt.Sprintf("100%s,100%s", "coin-4", "coin-5"),
		/*swapFee=*/ "0.01",
		/*exitFee=*/ "0.01",
		/*poolType=*/ "balancer",
		/*amplification=*/ "0",
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.WaitForNextBlock())

	resp := &sdk.TxResponse{}
	val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
	resp, err = testutilcli.QueryTx(s.network.Validators[0].ClientCtx, resp.TxHash)
	s.Require().NoError(err)

	poolID, err := ExtractPoolIDFromCreatePoolResponse(val.ClientCtx.Codec, resp)
	s.Require().NoError(err, out.String())

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
			tokenOutDenom: "uusdc",
			expectErr:     true,
		},
		{
			name:          "invalid token in",
			poolId:        poolID,
			tokenIn:       "0coin-4",
			tokenOutDenom: "uusdc",
			expectErr:     true,
		},
		{
			name:          "invalid token out denom",
			poolId:        poolID,
			tokenIn:       "50coin-4",
			tokenOutDenom: "",
			expectErr:     true,
		},
		{
			name:          "pool not found",
			poolId:        1000000,
			tokenIn:       "50unibi",
			tokenOutDenom: "uusdc",
			respType:      &sdk.TxResponse{},
			expectedCode:  types.ErrPoolNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token in denom not found",
			poolId:        poolID,
			tokenIn:       "50foo",
			tokenOutDenom: "coin-5",
			respType:      &sdk.TxResponse{},
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token out denom not found",
			poolId:        poolID,
			tokenIn:       "50coin-4",
			tokenOutDenom: "foo",
			respType:      &sdk.TxResponse{},
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "successful swap",
			poolId:        poolID,
			tokenIn:       "50coin-4",
			tokenOutDenom: "coin-5",
			respType:      &sdk.TxResponse{},
			expectedCode:  0,
			expectErr:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		ctx := val.ClientCtx

		s.Run(tc.name, func() {
			out, err := ExecMsgSwapAssets(ctx, tc.poolId, val.Address, tc.tokenIn, tc.tokenOutDenom)
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

func (s *IntegrationTestSuite) TestSwapStableAssets() {
	s.T().SkipNow()
	val := s.network.Validators[0]

	// create a new pool
	out, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ fmt.Sprintf("1%s,1%s", "coin-1", "coin-5"),
		/*tokenWeights=*/ fmt.Sprintf("100%s,100%s", "coin-1", "coin-5"),
		/*swapFee=*/ "0.01",
		/*exitFee=*/ "0.01",
		/*poolType=*/ "stableswap",
		/*amplification=*/ "42",
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.WaitForNextBlock())

	resp := &sdk.TxResponse{}
	val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
	resp, err = testutilcli.QueryTx(s.network.Validators[0].ClientCtx, resp.TxHash)
	s.Require().NoError(err)

	poolID, err := ExtractPoolIDFromCreatePoolResponse(val.ClientCtx.Codec, resp)
	s.Require().NoError(err, out.String())

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
			tokenOutDenom: "uusdc",
			expectErr:     true,
		},
		{
			name:          "invalid token in",
			poolId:        poolID,
			tokenIn:       "0coin-1",
			tokenOutDenom: "uusdc",
			expectErr:     true,
		},
		{
			name:          "invalid token out denom",
			poolId:        poolID,
			tokenIn:       "50coin-1",
			tokenOutDenom: "",
			expectErr:     true,
		},
		{
			name:          "pool not found",
			poolId:        1000000,
			tokenIn:       "50unibi",
			tokenOutDenom: "uusdc",
			respType:      &sdk.TxResponse{},
			expectedCode:  types.ErrPoolNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token in denom not found",
			poolId:        poolID,
			tokenIn:       "50foo",
			tokenOutDenom: "coin-5",
			respType:      &sdk.TxResponse{},
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token out denom not found",
			poolId:        poolID,
			tokenIn:       "50coin-1",
			tokenOutDenom: "foo",
			respType:      &sdk.TxResponse{},
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "successful swap",
			poolId:        poolID,
			tokenIn:       "50coin-1",
			tokenOutDenom: "coin-5",
			respType:      &sdk.TxResponse{},
			expectedCode:  0,
			expectErr:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		ctx := val.ClientCtx

		s.Run(tc.name, func() {
			out, err := ExecMsgSwapAssets(ctx, tc.poolId, val.Address, tc.tokenIn, tc.tokenOutDenom)
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

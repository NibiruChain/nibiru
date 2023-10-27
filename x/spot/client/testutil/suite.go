package testutil

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/x/common/testutil"
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
	testutil.BeforeIntegrationSuite(s.T())

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

func (s *IntegrationTestSuite) TestCreatePoolCmd() {
	val := s.network.Validators[0]

	tc := []struct {
		name           string
		tokenWeights   string
		initialDeposit string
		poolType       string
		amplification  string

		expectedErr  error
		expectedCode uint32
	}{
		{
			name:           "create pool with insufficient funds",
			tokenWeights:   fmt.Sprintf("1%s, 1%s", "coin-1", "coin-2"),
			initialDeposit: fmt.Sprintf("1000000000%s,10000000000%s", "coin-1", "coin-2"),
			poolType:       "balancer",
			amplification:  "0",
			expectedCode:   5, // bankKeeper code for insufficient funds
		},
		{
			name:           "create pool with invalid weights",
			tokenWeights:   fmt.Sprintf("0%s, 1%s", "coin-1", "coin-2"),
			initialDeposit: fmt.Sprintf("10000%s,10000%s", "coin-1", "coin-2"),
			poolType:       "balancer",
			amplification:  "0",
			expectedErr:    types.ErrInvalidCreatePoolArgs,
		},
		{
			name:           "create pool with deposit not matching weights",
			tokenWeights:   fmt.Sprintf("1%s, 1%s", "coin-1", "coin-2"),
			initialDeposit: "1000foo,10000uusdc",
			poolType:       "balancer",
			amplification:  "0",
			expectedErr:    types.ErrInvalidCreatePoolArgs,
		},
		{
			name:           "create a stableswap pool, amplification parameter below 1",
			tokenWeights:   fmt.Sprintf("1%s, 1%s", "coin-1", "coin-2"),
			initialDeposit: fmt.Sprintf("1000%s,10000%s", "coin-1", "coin-2"),
			amplification:  "0",
			poolType:       "stableswap",
			expectedErr:    types.ErrAmplificationTooLow,
		},
		{
			name:           "happy path - balancer",
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
			out, err := ExecMsgCreatePool(
				s.T(),
				val.ClientCtx,
				val.Address,
				tc.tokenWeights,
				tc.initialDeposit,
				"0.003",
				"0.003",
				tc.poolType,
				tc.amplification,
			)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr, out.String())
			} else {
				s.Require().NoError(err)
				s.Require().NoError(s.network.WaitForNextBlock())

				txResp := sdk.TxResponse{}
				val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), &txResp)
				resp, err := testutilcli.QueryTx(val.ClientCtx, txResp.TxHash)
				s.Require().NoError(err)

				s.Assert().Equal(tc.expectedCode, resp.Code, string(val.ClientCtx.Codec.MustMarshalJSON(resp)))
			}
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
	val := s.network.Validators[0]

	// create a new pool
	out, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ fmt.Sprintf("1%s,1%s", "coin-3", "coin-4"),
		/*initialDeposit=*/ fmt.Sprintf("100%s,100%s", "coin-3", "coin-4"),
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
		poolSharesOut string

		expectErr    bool
		expectedCode uint32

		expectedCoin3 sdkmath.Int
		expectedCoin4 sdkmath.Int
	}{
		{
			name:          "exit pool from invalid pool",
			poolId:        100,
			poolSharesOut: "100nibiru/pool/100",
			expectErr:     false,
			expectedCode:  1, // spot.types.ErrNonExistingPool
			expectedCoin3: sdk.ZeroInt(),
			expectedCoin4: sdk.ZeroInt(),
		},
		{
			name:          "exit pool for too many shares",
			poolId:        poolID,
			poolSharesOut: fmt.Sprintf("1001000000000000000000nibiru/pool/%d", poolID),
			expectErr:     false,
			expectedCode:  1,
			expectedCoin3: sdk.ZeroInt(),
			expectedCoin4: sdk.ZeroInt(),
		},
		{
			name:          "exit pool for zero shares",
			poolId:        poolID,
			poolSharesOut: fmt.Sprintf("0nibiru/pool/%d", poolID),
			expectErr:     false,
			expectedCode:  1,
			expectedCoin3: sdk.ZeroInt(),
			expectedCoin4: sdk.ZeroInt(),
		},
		{ // Looks with a bug
			name:          "exit pool with sufficient balance",
			poolId:        poolID,
			poolSharesOut: fmt.Sprintf("100000000000000000000nibiru/pool/%d", poolID),
			expectErr:     false,
			expectedCode:  0,
			expectedCoin3: sdk.NewInt(99), // Received coin-3 minus 1 exit pool fee
			expectedCoin4: sdk.NewInt(99), // Received coin-4 minus 1 exit pool fee
		},
	}

	for _, tc := range testCases {
		tc := tc
		ctx := val.ClientCtx

		s.Run(tc.name, func() {
			// Get original balance
			resp, err := sdktestutil.QueryBalancesExec(ctx, val.Address)
			s.Require().NoError(err)
			var originalBalance banktypes.QueryAllBalancesResponse
			ctx.Codec.MustUnmarshalJSON(resp.Bytes(), &originalBalance)

			out, err := ExecMsgExitPool(ctx, tc.poolId, val.Address, tc.poolSharesOut)

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

				// Ensure balance is ok
				out, err = sdktestutil.QueryBalancesExec(ctx, val.Address)
				s.Require().NoError(err)
				var finalBalance banktypes.QueryAllBalancesResponse
				ctx.Codec.MustUnmarshalJSON(out.Bytes(), &finalBalance)

				s.Assert().Equal(
					originalBalance.Balances.AmountOf("coin-3").Add(tc.expectedCoin3),
					finalBalance.Balances.AmountOf("coin-3"),
				)
				s.Assert().Equal(
					originalBalance.Balances.AmountOf("coin-4").Add(tc.expectedCoin4),
					finalBalance.Balances.AmountOf("coin-4"),
				)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewExitStablePoolCmd() {
	val := s.network.Validators[0]

	// create a new pool
	out, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ fmt.Sprintf("1%s,1%s", "coin-3", "coin-5"),
		/*tokenWeights=*/ fmt.Sprintf("100%s,100%s", "coin-3", "coin-5"),
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
	s.Require().NoErrorf(err, "cmd output: %s", out)

	poolID, err := ExtractPoolIDFromCreatePoolResponse(val.ClientCtx.Codec, resp)
	s.Require().NoErrorf(err, "cmd output: %s", out)

	testCases := []struct {
		name          string
		poolId        uint64
		poolSharesOut string
		expectErr     bool
		expectedCode  uint32
		expectedCoin3 sdkmath.Int
		expectedCoin5 sdkmath.Int
	}{
		{
			name:          "exit pool from invalid pool",
			poolId:        100,
			poolSharesOut: "100nibiru/pool/100",
			expectErr:     false,
			expectedCode:  1, // spot.types.ErrNonExistingPool
			expectedCoin3: sdk.ZeroInt(),
			expectedCoin5: sdk.ZeroInt(),
		},
		{
			name:          "exit pool for too many shares",
			poolId:        poolID,
			poolSharesOut: fmt.Sprintf("1001000000000000000000nibiru/pool/%d", poolID),
			expectErr:     false,
			expectedCode:  1,
			expectedCoin3: sdk.ZeroInt(),
			expectedCoin5: sdk.ZeroInt(),
		},
		{
			name:          "exit pool for zero shares",
			poolId:        poolID,
			poolSharesOut: fmt.Sprintf("0nibiru/pool/%d", poolID),
			expectErr:     false,
			expectedCode:  1,
			expectedCoin3: sdk.ZeroInt(),
			expectedCoin5: sdk.ZeroInt(),
		},
		{ // Looks with a bug
			name:          "exit pool with sufficient balance",
			poolId:        poolID,
			poolSharesOut: fmt.Sprintf("100000000000000000000nibiru/pool/%d", poolID),
			expectErr:     false,
			expectedCode:  0,
			expectedCoin3: sdk.NewInt(99), // Received coin-3 minus 1 exit pool fee
			expectedCoin5: sdk.NewInt(99), // Received coin-5 minus 1 exit pool fee
		},
	}

	for _, tc := range testCases {
		tc := tc
		ctx := val.ClientCtx

		s.Run(tc.name, func() {
			// Get original balance
			resp, err := sdktestutil.QueryBalancesExec(ctx, val.Address)
			s.Require().NoError(err)
			var originalBalance banktypes.QueryAllBalancesResponse
			ctx.Codec.MustUnmarshalJSON(resp.Bytes(), &originalBalance)

			out, err := ExecMsgExitPool(ctx, tc.poolId, val.Address, tc.poolSharesOut)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(s.network.WaitForNextBlock())

				resp := &sdk.TxResponse{}
				ctx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
				resp, err = testutilcli.QueryTx(ctx, resp.TxHash)
				s.Require().NoError(err)

				s.Require().Equal(tc.expectedCode, resp.Code, out.String())

				// Ensure balance is ok
				out, err := sdktestutil.QueryBalancesExec(ctx, val.Address)
				s.Require().NoError(err)
				var finalBalance banktypes.QueryAllBalancesResponse
				ctx.Codec.MustUnmarshalJSON(out.Bytes(), &finalBalance)

				s.Assert().Equal(
					originalBalance.Balances.AmountOf("coin-3").Add(tc.expectedCoin3),
					finalBalance.Balances.AmountOf("coin-3"),
				)
				s.Assert().Equal(
					originalBalance.Balances.AmountOf("coin-5").Add(tc.expectedCoin5),
					finalBalance.Balances.AmountOf("coin-5"),
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

			out, err := sdktestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
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
			expectedCode:  types.ErrPoolNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token in denom not found",
			poolId:        poolID,
			tokenIn:       "50foo",
			tokenOutDenom: "coin-5",
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token out denom not found",
			poolId:        poolID,
			tokenIn:       "50coin-4",
			tokenOutDenom: "foo",
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "successful swap",
			poolId:        poolID,
			tokenIn:       "50coin-4",
			tokenOutDenom: "coin-5",
			expectedCode:  0,
			expectErr:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			out, err := ExecMsgSwapAssets(val.ClientCtx, tc.poolId, val.Address, tc.tokenIn, tc.tokenOutDenom)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(s.network.WaitForNextBlock())

				resp := &sdk.TxResponse{}
				val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
				resp, err = testutilcli.QueryTx(val.ClientCtx, resp.TxHash)
				s.Require().NoError(err)

				s.Assert().Equal(tc.expectedCode, resp.Code, out.String())
			}
		})
	}
}

func (s *IntegrationTestSuite) TestSwapStableAssets() {
	s.Require().NoError(s.network.WaitForNextBlock())
	val := s.network.Validators[0]

	// create a new pool
	out, err := ExecMsgCreatePool(
		s.T(),
		val.ClientCtx,
		/*owner-*/ val.Address,
		/*tokenWeights=*/ fmt.Sprintf("1%s,1%s", "coin-1", "coin-5"),
		/*initialDeposit=*/ fmt.Sprintf("100%s,100%s", "coin-1", "coin-5"),
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
			expectedCode:  types.ErrPoolNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token in denom not found",
			poolId:        poolID,
			tokenIn:       "50foo",
			tokenOutDenom: "coin-5",
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "token out denom not found",
			poolId:        poolID,
			tokenIn:       "50coin-1",
			tokenOutDenom: "foo",
			expectedCode:  types.ErrTokenDenomNotFound.ABCICode(),
			expectErr:     false,
		},
		{
			name:          "successful swap",
			poolId:        poolID,
			tokenIn:       "50coin-1",
			tokenOutDenom: "coin-5",
			expectedCode:  0,
			expectErr:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			out, err := ExecMsgSwapAssets(val.ClientCtx, tc.poolId, val.Address, tc.tokenIn, tc.tokenOutDenom)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(s.network.WaitForNextBlock())

				resp := &sdk.TxResponse{}
				val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), resp)
				resp, err = testutilcli.QueryTx(val.ClientCtx, resp.TxHash)
				s.Require().NoError(err)

				s.Assert().Equal(tc.expectedCode, resp.Code, out.String())
			}
		})
	}
}

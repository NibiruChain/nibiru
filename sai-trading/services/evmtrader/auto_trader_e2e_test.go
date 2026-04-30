//go:build e2e
// +build e2e

package evmtrader_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/sai-trading/services/evmtrader"
	"github.com/NibiruChain/nibiru/sai-trading/tutil"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/gosdk"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const localnetValidatorMnemonic = "guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"

type AutoTradingE2ETestSuite struct {
	suite.Suite
	ctx           context.Context
	trader        *evmtrader.EVMTrader
	config        evmtrader.Config
	initialBlock  uint64
	testStartTime time.Time
}

func (s *AutoTradingE2ETestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.testStartTime = time.Now()

	// Ensure local blockchain is running
	if err := tutil.EnsureLocalBlockchain(); err != nil {
		require.Fail(s.T(), "Local blockchain is not running", "error: %v", err)
		return
	}
	s.T().Logf("✓ Local blockchain is running")

	cfg := evmtrader.Config{
		EVMRPCUrl:       "http://localhost:8545",
		GrpcUrl:         "localhost:9090",
		ChainID:         "nibiru-localnet-0",
		CollateralIndex: 1,
	}

	contractsEnv := "../../.cache/localnet_contracts.env"
	cfg.ContractsEnvFile = contractsEnv

	cfg.Mnemonic = localnetValidatorMnemonic

	s.config = cfg

	trader, err := evmtrader.New(s.ctx, cfg)
	require.NoError(s.T(), err, "Failed to create EVMTrader")
	s.trader = trader

	s.fundTestAccountWithStNIBI()

	initialBlock, err := s.trader.Client().BlockNumber(s.ctx)
	require.NoError(s.T(), err, "Failed to get initial block number")
	s.initialBlock = initialBlock

	s.T().Logf("✓ Test suite setup complete")
	s.T().Logf("  - Network: %s", cfg.EVMRPCUrl)
	s.T().Logf("  - Initial block: %d", initialBlock)
	s.T().Logf("  - Account: %s", trader.AccountAddr().Hex())
}

func (s *AutoTradingE2ETestSuite) TearDownSuite() {
	if s.trader != nil {
		s.trader.Close()
	}

	duration := time.Since(s.testStartTime)
	s.T().Logf("✓ Test suite completed in %v", duration)
}

func (s *AutoTradingE2ETestSuite) SetupTest() {
	s.T().Logf("\n━━━ Starting: %s ━━━", s.T().Name())
}

func (s *AutoTradingE2ETestSuite) TearDownTest() {
	s.cleanupTestPositions()
	s.T().Logf("━━━ Completed: %s ━━━\n", s.T().Name())
}

func (s *AutoTradingE2ETestSuite) TestAutoTrading_Basic() {
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Minute)
	defer cancel()

	autoCfg := evmtrader.AutoTradingConfig{
		MarketIndices:     []uint64{0},
		CollateralIndices: []uint64{s.config.CollateralIndex},
		MinTradeSize:      100_000,
		MaxTradeSize:      300_000,
		MinLeverage:       1,
		MaxLeverage:       2,
		BlocksBeforeClose: 5,
		MaxOpenPositions:  2,
		LoopDelaySeconds:  3,
	}

	tradingCtx, tradingCancel := context.WithTimeout(ctx, 90*time.Second)
	defer tradingCancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.trader.RunAutoTrading(tradingCtx, autoCfg)
	}()

	s.T().Log("✓ Auto-trading started")

	s.T().Log("Waiting for positions to open...")
	time.Sleep(15 * time.Second)

	trades, err := s.trader.QueryTrades(ctx)
	require.NoError(s.T(), err, "Failed to query trades")

	openPositions := s.filterOpenPositions(trades)
	s.T().Logf("Found %d open positions", len(openPositions))

	// Verify: Should have opened at least 1 position
	require.Greater(s.T(), len(openPositions), 0, "Should have opened at least 1 position")

	// Verify: Should not exceed max positions
	require.LessOrEqual(s.T(), len(openPositions), autoCfg.MaxOpenPositions,
		"Should not exceed MaxOpenPositions")

	// Track first position
	firstTradeIndex := openPositions[0].UserTradeIndex
	s.T().Logf("Tracking position %d", firstTradeIndex)

	// Wait for position to close
	s.T().Log("Waiting for positions to close...")
	sleepTime := time.Duration(autoCfg.BlocksBeforeClose+2) * 10 * time.Second
	s.T().Logf("Sleeping for %v", sleepTime)
	time.Sleep(sleepTime)

	// Verify: Position should be closed
	trades, err = s.trader.QueryTrades(ctx)
	require.NoError(s.T(), err)

	positionClosed := true
	for _, trade := range trades {
		if trade.UserTradeIndex == firstTradeIndex && trade.IsOpen {
			positionClosed = false
			break
		}
	}

	s.T().Logf("Position %d closed: %v", firstTradeIndex, positionClosed)
	require.True(s.T(), positionClosed, "Position should be closed after BlocksBeforeClose")

	tradingCancel()
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			s.T().Logf("Auto-trading stopped with: %v", err)
		}
	case <-time.After(5 * time.Second):
		s.T().Log("Auto-trading shutdown timed out")
	}

	s.T().Log("✓ Test completed successfully")
}

func (s *AutoTradingE2ETestSuite) filterOpenPositions(trades []evmtrader.ParsedTrade) []evmtrader.ParsedTrade {
	open := make([]evmtrader.ParsedTrade, 0)
	for _, trade := range trades {
		if trade.IsOpen {
			open = append(open, trade)
		}
	}
	return open
}

func (s *AutoTradingE2ETestSuite) cleanupTestPositions() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	trades, err := s.trader.QueryTrades(ctx)
	if err != nil {
		s.T().Logf("Failed to query trades for cleanup: %v", err)
		return
	}

	openPositions := s.filterOpenPositions(trades)
	if len(openPositions) == 0 {
		return
	}

	s.T().Logf("Cleaning up %d open positions...", len(openPositions))

	for _, trade := range openPositions {
		if err := s.trader.CloseTrade(ctx, trade.UserTradeIndex); err != nil {
			s.T().Logf("Failed to close position %d: %v", trade.UserTradeIndex, err)
		} else {
			s.T().Logf("  ✓ Closed position %d", trade.UserTradeIndex)
		}
		time.Sleep(2 * time.Second) // Avoid nonce conflicts
	}
}

func (s *AutoTradingE2ETestSuite) fundTestAccountWithStNIBI() {
	evmAddr := s.trader.AccountAddr()
	nibiruAddr := eth.EthAddrToNibiruAddr(evmAddr)
	nibiruAddrBech32 := nibiruAddr.String()

	collateralIndex := s.config.CollateralIndex
	if collateralIndex == 0 {
		return
	}

	collateralDenom, err := s.trader.QueryCollateralDenom(s.ctx, collateralIndex)
	if err != nil {
		s.T().Logf("Failed to query collateral denom for index %d: %v", collateralIndex, err)
		return
	}
	if collateralDenom == "" {
		return
	}

	// Check balance first - skip if already funded
	bankClient := banktypes.NewQueryClient(s.trader.GRPCConn())
	resp, err := bankClient.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: nibiruAddrBech32,
		Denom:   collateralDenom,
	})
	if err == nil && resp.Balance != nil {
		balance := resp.Balance.Amount.BigInt()
		minRequired := big.NewInt(1000000)
		if balance.Cmp(minRequired) >= 0 {
			return // Already has sufficient balance
		}
	}

	// Use programmatic SDK to fund account
	netInfo := gosdk.NETWORK_INFO_DEFAULT
	grpcConn, err := gosdk.GetGRPCConnection(netInfo.GrpcEndpoint, true, 10)
	if err != nil {
		s.T().FailNow()
		return
	}
	defer grpcConn.Close()

	nibiruSdk, err := gosdk.NewNibiruSdk(s.config.ChainID, grpcConn, netInfo.TmRpcEndpoint)
	if err != nil {
		s.T().FailNow()
		return
	}

	// Add validator to keyring
	validatorAddr, err := gosdk.AddSignerToKeyringSecp256k1(nibiruSdk.Keyring, localnetValidatorMnemonic, "validator")
	if err != nil {
		s.T().FailNow()
		return
	}

	// Create MsgSend
	toAddr, err := sdk.AccAddressFromBech32(nibiruAddrBech32)
	if err != nil {
		s.T().FailNow()
		return
	}

	amount := sdk.NewIntFromUint64(100000000)
	coins := sdk.NewCoins(sdk.NewCoin(collateralDenom, amount))
	msg := banktypes.NewMsgSend(validatorAddr, toAddr, coins)

	// Broadcast transaction
	txResp, err := nibiruSdk.BroadcastMsgsGrpc(validatorAddr, msg)
	if err != nil {
		s.T().FailNow()
		return
	}

	if txResp.Code != 0 {
		s.T().FailNow()
		return
	}

	// Wait for confirmation
	time.Sleep(5 * time.Second)

	// Verify balance
	resp, err = bankClient.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: nibiruAddrBech32,
		Denom:   collateralDenom,
	})
	if err != nil {
		s.T().FailNow()
		return
	}

	if resp.Balance == nil {
		s.T().FailNow()
		return
	}

	balance := resp.Balance.Amount.BigInt()
	if balance.Cmp(big.NewInt(0)) == 0 {
		s.T().FailNow()
		return
	}
}

// TestAutoTradingE2ESuite runs the E2E test suite
func TestAutoTradingE2ESuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	suite.Run(t, new(AutoTradingE2ETestSuite))
}

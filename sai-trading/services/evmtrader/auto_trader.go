package evmtrader

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// AutoTradingConfig holds configuration for automated trading
type AutoTradingConfig struct {
	MarketIndex       uint64
	CollateralIndex   uint64
	MinTradeSize      uint64
	MaxTradeSize      uint64
	MinLeverage       uint64
	MaxLeverage       uint64
	BlocksBeforeClose uint64
	MaxOpenPositions  int
	LoopDelaySeconds  int
}

// PositionTracker tracks an open position and when it was opened
type PositionTracker struct {
	TradeIndex  uint64
	OpenBlock   uint64
	MarketIndex uint64
}

// RunAutoTrading runs the automated trading loop
func (t *EVMTrader) RunAutoTrading(ctx context.Context, cfg AutoTradingConfig) error {
	// Initialize token denom map at start for better logging
	if err := t.InitializeTokenDenomMap(ctx, cfg.MarketIndex); err != nil {
		t.logWarn("Failed to initialize token denom map", "error", err.Error())
	}

	var (
		baseIndex, quoteIndex *uint64
		baseDenom, quoteDenom string
	)
	if market, err := t.queryMarket(ctx, cfg.MarketIndex); err != nil {
		t.logWarn("Failed to query market for pair info", "market_index", cfg.MarketIndex, "error", err.Error())
	} else {
		baseIndex = market.BaseToken
		quoteIndex = market.QuoteToken
		if baseIndex != nil {
			baseDenom = t.GetTokenDenom(*baseIndex)
		}
		if quoteIndex != nil {
			quoteDenom = t.GetTokenDenom(*quoteIndex)
		}
	}

	collateralDenom := t.GetCollateralDenom(cfg.CollateralIndex)

	t.logInfo("Starting automated trading",
		"market_index", cfg.MarketIndex,
		"base_denom", baseDenom,
		"quote_denom", quoteDenom,
		"collateral_denom", collateralDenom,
		"min_trade_size", cfg.MinTradeSize,
		"max_trade_size", cfg.MaxTradeSize,
		"min_leverage", cfg.MinLeverage,
		"max_leverage", cfg.MaxLeverage,
		"blocks_before_close", cfg.BlocksBeforeClose,
		"max_open_positions", cfg.MaxOpenPositions,
	)

	// Map to track positions we've opened (tradeIndex -> PositionTracker)
	trackedPositions := make(map[uint64]*PositionTracker)

	for {
		// Get current block number
		currentBlock, err := t.client.BlockNumber(ctx)
		if err != nil {
			t.logError("Failed to get block number", "error", err.Error())
			time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
			continue
		}

		t.logDebug("Auto-trading loop iteration", "current_block", currentBlock, "tracked_positions", len(trackedPositions))

		// Query current open positions
		trades, err := t.QueryTrades(ctx)
		if err != nil {
			t.logError("Failed to query trades", "error", err.Error())
			time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
			continue
		}

		// Filter for open positions
		openPositions := make([]ParsedTrade, 0)
		for _, trade := range trades {
			if trade.IsOpen {
				openPositions = append(openPositions, trade)
			}
		}

		t.logInfo("Found open positions", "count", len(openPositions))

		// Check if any tracked positions should be closed
		// Only close one position per block iteration to avoid nonce conflicts
		closedOne := false
		for _, trade := range openPositions {
			tracker, isTracked := trackedPositions[trade.UserTradeIndex]
			if !isTracked {
				// Position not tracked (may have been opened before bot started)
				// Add it to tracking with current block as open block
				trackedPositions[trade.UserTradeIndex] = &PositionTracker{
					TradeIndex:  trade.UserTradeIndex,
					OpenBlock:   currentBlock,
					MarketIndex: trade.MarketIndex,
				}
				t.logDebug("Added existing position to tracking", "trade_index", trade.UserTradeIndex, "current_block", currentBlock)
				continue
			}

			// Check if position should be closed
			blocksSinceOpen := currentBlock - tracker.OpenBlock
			if blocksSinceOpen >= cfg.BlocksBeforeClose {
				// Only close one position per block iteration
				if closedOne {
					continue
				}

				t.logInfo("Closing position (reached block threshold)",
					"trade_index", trade.UserTradeIndex,
					"blocks_since_open", blocksSinceOpen,
					"threshold", cfg.BlocksBeforeClose,
				)

				if err := t.CloseTrade(ctx, trade.UserTradeIndex); err != nil {
					t.logError("Failed to close trade", "trade_index", trade.UserTradeIndex, "error", err.Error())
				} else {
					// Remove from tracking
					delete(trackedPositions, trade.UserTradeIndex)
					// Wait 2 seconds after closing to ensure nonce is updated before next operation
					time.Sleep(2 * time.Second)
				}

				// Mark that we've closed one position this iteration
				closedOne = true
				break
			}
		}

		// Check if we should open a new position
		// Only open if we didn't close a position in this iteration (to avoid nonce conflicts)
		if !closedOne && len(openPositions) < cfg.MaxOpenPositions {
			// Generate random trade parameters
			leverage := randomUint64(cfg.MinLeverage, cfg.MaxLeverage)
			isLong := randomBool()
			tradeType := randomTradeType()

			collateralIndex := cfg.CollateralIndex
			if collateralIndex == 0 {
				market, err := t.queryMarket(ctx, cfg.MarketIndex)
				if err != nil {
					t.logError("Failed to query market for collateral index", "error", err.Error())
					time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
					continue
				}
				if market.QuoteToken == nil {
					t.logError("Market has no quote token", "market_index", cfg.MarketIndex)
					time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
					continue
				}
				collateralIndex = *market.QuoteToken
			}

			collateralDenom, err := t.queryCollateralDenom(ctx, collateralIndex)
			if err != nil {
				t.logError("Failed to query collateral denom",
					"collateral_index", collateralIndex,
					"error", err.Error(),
				)
				time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
				continue
			}

			balance, err := t.queryCosmosBalance(ctx, t.ethAddrBech32, collateralDenom)
			if err != nil {
				t.logError("Failed to query balance",
					"denom", collateralDenom,
					"error", err.Error(),
				)
				time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
				continue
			}

			// Estimate gas and check balance for gas fees
			gasLimit := uint64(2_000_000)
			gasPrice, err := t.client.SuggestGasPrice(ctx)
			if err != nil {
				t.logWarn("Failed to get gas price, using default", "error", err.Error())
				gasPrice = big.NewInt(1000)
			}

			estimatedGasCost := new(big.Int).Mul(big.NewInt(int64(gasLimit)), gasPrice)

			evmBalance, err := t.client.BalanceAt(ctx, t.accountAddr, nil)
			if err != nil {
				t.logError("Failed to query EVM balance for gas check", "error", err.Error())
				time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
				continue
			}

			if evmBalance.Cmp(estimatedGasCost) < 0 {
				balanceNibi := new(big.Float).Quo(new(big.Float).SetInt(evmBalance), big.NewFloat(1e18))
				t.logError("Insufficient NIBI balance for gas fees",
					"balance_nibi", balanceNibi.Text('f', 18),
					"balance_wei", evmBalance.String(),
					"gas_limit", gasLimit,
					"gas_price", gasPrice.String(),
					"fund this address", fmt.Sprintf(" %s with %s", t.ethAddrBech32, "unibi"),
				)
				return fmt.Errorf("insufficient balance")
			}

			if balance.Cmp(big.NewInt(0)) == 0 {
				t.logError("Balance is zero, stopping automated trading",
					"balance", balance.String(),
					"collateral_denom", collateralDenom,
					"fund_this_address", fmt.Sprintf("%s with %s", t.ethAddrBech32, collateralDenom),
				)
				return fmt.Errorf("balance is zero for collateral denom %s, fund this address %s", collateralDenom, t.ethAddrBech32)
			}

			collateralAmount := randomUint64(cfg.MinTradeSize, cfg.MaxTradeSize)
			collateralAmtBig := new(big.Int).SetUint64(collateralAmount)

			if balance.Cmp(collateralAmtBig) < 0 {
				t.logError("Insufficient balance",
					"balance", balance.String(),
					"required", collateralAmtBig.String(),
					"collateral_denom", collateralDenom,
					"fund_this_address", fmt.Sprintf("%s with %s", t.ethAddrBech32, collateralDenom),
				)
				time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
				continue
			}

			tradeSize := collateralAmount * leverage

			// Open the trade
			chainID, err := t.client.ChainID(ctx)
			if err != nil {
				t.logError("Failed to get chain ID", "error", err.Error())
				time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
				continue
			}

			// Fetch current market price from oracle (needed for all trade types)
			marketPrice, err := t.fetchMarketPriceForIndex(ctx, cfg.MarketIndex)
			if err != nil {
				t.logError("Failed to fetch market price from oracle", "error", err.Error())
				time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
				continue
			}

			// Determine open_price based on trade type
			var openPrice *float64
			if tradeType == TradeTypeMarket {
				// For market orders, use current market price
				openPrice = &marketPrice
			} else {
				// For limit/stop orders, adjust price by ±2-5% to create trigger price
				adjustmentPercent := randomFloat64(2.0, 5.0) / 100.0
				if isLong {
					// For long limit orders, set trigger price below market (buy cheaper)
					// For long stop orders, set trigger price below market (stop loss when price drops)
					if tradeType == TradeTypeLimit {
						triggerPrice := marketPrice * (1.0 - adjustmentPercent)
						openPrice = &triggerPrice
					} else { // stop
						triggerPrice := marketPrice * (1.0 - adjustmentPercent)
						openPrice = &triggerPrice
					}
				} else {
					// For short limit orders, set trigger price above market (sell higher)
					// For short stop orders, set trigger price above market (stop loss when price rises)
					if tradeType == TradeTypeLimit {
						triggerPrice := marketPrice * (1.0 + adjustmentPercent)
						openPrice = &triggerPrice
					} else { // stop
						triggerPrice := marketPrice * (1.0 + adjustmentPercent)
						openPrice = &triggerPrice
					}
				}
			}

			params := &OpenTradeParams{
				MarketIndex:     cfg.MarketIndex,
				Leverage:        leverage,
				Long:            isLong,
				CollateralIndex: collateralIndex,
				TradeType:       tradeType,
				OpenPrice:       openPrice,
				TP:              nil,
				SL:              nil,
				SlippageP:       "1",
				CollateralAmt:   collateralAmtBig,
			}

			t.logInfo("Opening new random position",
				"current_positions", len(openPositions),
				"max", cfg.MaxOpenPositions,
				"collateral", collateralAmount,
				"leverage", leverage,
				"trade_size", tradeSize,
				"long", isLong,
				"trade_type", tradeType,
				"market_index", cfg.MarketIndex,
				"collateral_index", collateralIndex,
				"open_price", *openPrice,
			)

			if err := t.OpenTrade(ctx, chainID, params); err != nil {
				t.logError("Failed to open trade",
					"error", err.Error(),
					"trade_type", tradeType,
					"leverage", leverage,
					"long", isLong,
					"trade_size", tradeSize,
					"market_index", cfg.MarketIndex,
				)
				// If it's a nonce error, wait a bit longer before retrying
				if strings.Contains(err.Error(), "invalid nonce") {
					t.logError("Nonce conflict detected",
						"error", err.Error(),
						"action", "waiting before next iteration",
					)
					time.Sleep(3 * time.Second)
				}
			} else {
				// Query trades again to find the new position and add it to tracking
				newTrades, err := t.QueryTrades(ctx)
				if err != nil {
					t.logError("Failed to query trades after opening", "error", err.Error())
				} else {
					// Find the newest open position that we're not tracking yet
					for _, trade := range newTrades {
						if trade.IsOpen && trackedPositions[trade.UserTradeIndex] == nil {
							trackedPositions[trade.UserTradeIndex] = &PositionTracker{
								TradeIndex:  trade.UserTradeIndex,
								OpenBlock:   currentBlock,
								MarketIndex: trade.MarketIndex,
							}
							t.logDebug("Added new position to tracking",
								"trade_index", trade.UserTradeIndex,
								"open_block", currentBlock,
							)
							break
						}
					}
				}
				// Wait 2 seconds after successfully opening a trade to ensure nonce is updated
				time.Sleep(2 * time.Second)
			}
		} else {
			t.logInfo("Maximum open positions reached, waiting to close positions", "current", len(openPositions), "max", cfg.MaxOpenPositions)
		}

		// Sleep before next iteration
		time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
	}
}

// randomUint64 returns a random uint64 between min and max (inclusive)
func randomUint64(min, max uint64) uint64 {
	if min >= max {
		return min
	}
	diff := max - min + 1
	n, err := rand.Int(rand.Reader, big.NewInt(int64(diff)))
	if err != nil {
		// Fallback to min on error
		return min
	}
	return min + n.Uint64()
}

// randomBool returns a cryptographically secure random boolean
func randomBool() bool {
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return false
	}
	return b[0]&1 == 1
}

// randomTradeType returns a random trade type (market, limit, or stop)
func randomTradeType() string {
	// Randomly select between market (50%), limit (25%), and stop (25%)
	n := randomUint64(0, 3)
	switch n {
	case 0, 1:
		return TradeTypeMarket
	case 2:
		return TradeTypeLimit
	default:
		return TradeTypeStop
	}
}

// randomFloat64 returns a random float64 between min and max
func randomFloat64(min, max float64) float64 {
	if min >= max {
		return min
	}
	diff := max - min
	// Generate random bytes and convert to float64
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return min
	}
	// Convert bytes to uint64, then to float64 in range [0, 1)
	randUint64 := new(big.Int).SetBytes(b[:]).Uint64()
	randFloat := float64(randUint64) / float64(^uint64(0))
	return min + randFloat*diff
}

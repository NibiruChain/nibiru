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
	MarketIndices              []uint64
	CollateralIndices          []uint64
	MinTradeSize               uint64
	MaxTradeSize               uint64
	MinLeverage                uint64
	MaxLeverage                uint64
	BlocksBeforeClose          uint64
	MaxOpenPositions           int
	LoopDelaySeconds           int
	HealthCheckIntervalSeconds int // 0 = disabled; when > 0, log and Slack health summary every N seconds
	MinSecondsBetweenOpens     int // 0 = no throttle; when > 0, open at most once every N seconds (allows short loop_delay with rare opens)
}

// PositionTracker tracks an open position and when it was opened
type PositionTracker struct {
	TradeIndex  uint64
	OpenBlock   uint64
	MarketIndex uint64
}

// RunAutoTrading runs the automated trading loop
func (t *EVMTrader) RunAutoTrading(ctx context.Context, cfg AutoTradingConfig) error {
	return t.RunAutoTradingWithLoader(ctx, nil, cfg)
}

func (t *EVMTrader) RunAutoTradingWithLoader(ctx context.Context, loader *ConfigLoader, staticCfg AutoTradingConfig) error {
	var cfg AutoTradingConfig
	if loader != nil {
		cfg = loader.GetConfig()
		loader.StartWatcher(ctx)
	} else {
		cfg = staticCfg
	}

	marketIndices := cfg.MarketIndices
	collateralIndices := cfg.CollateralIndices

	// Initialize token denom map at start for better logging
	for _, marketIdx := range marketIndices {
		if err := t.InitializeTokenDenomMap(ctx, marketIdx); err != nil {
			t.logWarn("Failed to initialize token denom map", "market_index", marketIdx, "error", err.Error())
		}
	}

	marketPairs := make([]string, 0, len(marketIndices))
	for _, marketIdx := range marketIndices {
		market, err := t.queryMarket(ctx, marketIdx)
		if err == nil && market.BaseToken != nil && market.QuoteToken != nil {
			baseDenom := t.GetTokenDenom(*market.BaseToken)
			quoteDenom := t.GetTokenDenom(*market.QuoteToken)
			baseSymbol := extractSymbolFromDenom(baseDenom)
			quoteSymbol := extractSymbolFromDenom(quoteDenom)
			marketPairs = append(marketPairs, fmt.Sprintf("%s/%s", baseSymbol, quoteSymbol))
		} else {
			marketPairs = append(marketPairs, fmt.Sprintf("Market(%d)", marketIdx))
		}
	}

	collateralDenoms := make([]string, 0, len(collateralIndices))
	for _, collateralIdx := range collateralIndices {
		collateralDenom := t.GetCollateralDenom(collateralIdx)
		collateralSymbol := extractSymbolFromDenom(collateralDenom)
		collateralDenoms = append(collateralDenoms, collateralSymbol)
	}

	marketIndicesStr := formatUint64Slice(marketIndices)
	marketPairsStr := "[" + strings.Join(marketPairs, ", ") + "]"
	collateralIndicesStr := formatUint64Slice(collateralIndices)
	collateralDenomsStr := "[" + strings.Join(collateralDenoms, ", ") + "]"

	t.logInfo("Starting automated trading",
		"market_indices", marketIndicesStr,
		"market_pairs", marketPairsStr,
		"collateral_indices", collateralIndicesStr,
		"collateral_denoms", collateralDenomsStr,
		"min_trade_size", cfg.MinTradeSize,
		"max_trade_size", cfg.MaxTradeSize,
		"min_leverage", cfg.MinLeverage,
		"max_leverage", cfg.MaxLeverage,
		"blocks_before_close", cfg.BlocksBeforeClose,
		"max_open_positions", cfg.MaxOpenPositions,
		"loop_delay_seconds", cfg.LoopDelaySeconds,
		"min_seconds_between_opens", cfg.MinSecondsBetweenOpens,
	)

	// Map to track positions we've opened (tradeIndex -> PositionTracker)
	trackedPositions := make(map[uint64]*PositionTracker)
	var lastHealthCheck time.Time
	var lastOpenTime time.Time

	for {
		if loader != nil {
			cfg = loader.GetConfig()
			marketIndices = cfg.MarketIndices
			collateralIndices = cfg.CollateralIndices
		}

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

		// Periodic health check: log summary and optionally notify Slack
		if cfg.HealthCheckIntervalSeconds > 0 {
			interval := time.Duration(cfg.HealthCheckIntervalSeconds) * time.Second
			if lastHealthCheck.IsZero() || time.Since(lastHealthCheck) >= interval {
				t.runHealthCheck(ctx, cfg, currentBlock, openPositions, marketIndices, collateralIndices)
				lastHealthCheck = time.Now()
			}
		}

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
			// Throttle: don't open more than once every MinSecondsBetweenOpens (allows short loop_delay with rare opens)
			if cfg.MinSecondsBetweenOpens > 0 && !lastOpenTime.IsZero() && time.Since(lastOpenTime) < time.Duration(cfg.MinSecondsBetweenOpens)*time.Second {
				t.logDebug("Open cooldown active, skipping open",
					"elapsed_sec", int(time.Since(lastOpenTime).Seconds()),
					"min_seconds_between_opens", cfg.MinSecondsBetweenOpens,
				)
			} else {
				selectedMarketIndex := marketIndices[randomUint64(0, uint64(len(marketIndices)-1))]

				// Generate random trade parameters
				leverage := randomUint64(cfg.MinLeverage, cfg.MaxLeverage)
				isLong := randomBool()
				tradeType := randomTradeType()

				selectedCollateralIndex := collateralIndices[randomUint64(0, uint64(len(collateralIndices)-1))]

				collateralIndex := selectedCollateralIndex
				if collateralIndex == 0 {
					market, err := t.queryMarket(ctx, selectedMarketIndex)
					if err != nil {
						t.logError("Failed to query market for collateral index", "error", err.Error())
						time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
						continue
					}
					if market.QuoteToken == nil {
						t.logError("Market has no quote token", "market_index", selectedMarketIndex)
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
				marketPrice, err := t.fetchMarketPriceForIndex(ctx, selectedMarketIndex)
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
					MarketIndex:     selectedMarketIndex,
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

				// Get market pair info for nice logging
				market, err := t.queryMarket(ctx, selectedMarketIndex)
				var marketPair string
				if err == nil && market.BaseToken != nil && market.QuoteToken != nil {
					baseDenom := t.GetTokenDenom(*market.BaseToken)
					quoteDenom := t.GetTokenDenom(*market.QuoteToken)
					marketPair = fmt.Sprintf("%s/%s", baseDenom, quoteDenom)
				} else {
					marketPair = fmt.Sprintf("Market(%d)", selectedMarketIndex)
				}

				collateralDenomForLog := t.GetCollateralDenom(collateralIndex)
				collateralSymbol := extractSymbolFromDenom(collateralDenomForLog)
				collateralAmountFormatted := collateralAmtBig

				direction := "Short"
				if isLong {
					direction = "Long"
				}

				var openingLogMsg string
				switch tradeType {
				case TradeTypeMarket:
					openingLogMsg = fmt.Sprintf("Opening position: %s x%d %s, collateral %s %s",
						direction, leverage, marketPair, collateralAmountFormatted, collateralSymbol)
				case TradeTypeLimit:
					openingLogMsg = fmt.Sprintf("Opening limit order: %s x%d %s, collateral %s %s, trigger price: $%.2f",
						direction, leverage, marketPair, collateralAmountFormatted, collateralSymbol, *openPrice)
				default: // stop
					openingLogMsg = fmt.Sprintf("Opening stop order: %s x%d %s, collateral %s %s, trigger price: $%.2f",
						direction, leverage, marketPair, collateralAmountFormatted, collateralSymbol, *openPrice)
				}
				t.logInfo(openingLogMsg,
					"market_index", selectedMarketIndex,
					"collateral_index", collateralIndex,
				)

				if err := t.OpenTrade(ctx, chainID, params); err != nil {
					t.logError("Failed to open trade",
						"error", err.Error(),
						"trade_type", tradeType,
						"leverage", leverage,
						"long", isLong,
						"trade_size", tradeSize,
						"market_index", selectedMarketIndex,
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
					var logMsg string
					switch tradeType {
					case TradeTypeMarket:
						logMsg = fmt.Sprintf("Opened position: %s x%d %s, collateral %s %s",
							direction, leverage, marketPair, collateralAmountFormatted, collateralSymbol)
					case TradeTypeLimit:
						logMsg = fmt.Sprintf("Opened limit order: %s x%d %s, collateral %s %s, trigger price: $%.2f",
							direction, leverage, marketPair, collateralAmountFormatted, collateralSymbol, *openPrice)
					case TradeTypeStop: // stop
						logMsg = fmt.Sprintf("Opened stop order: %s x%d %s, collateral %s %s, trigger price: $%.2f",
							direction, leverage, marketPair, collateralAmountFormatted, collateralSymbol, *openPrice)
					default:
						t.logError("Unknown trade type", "trade_type", tradeType)
					}

					t.logInfo(logMsg,
						"current_positions", len(openPositions),
						"max", cfg.MaxOpenPositions,
						"market_index", selectedMarketIndex,
						"collateral_index", collateralIndex,
					)

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
					lastOpenTime = time.Now()
				}
			}
		} else {
			if closedOne {
				t.logInfo("Skipping iteration", "current", len(openPositions), "max", cfg.MaxOpenPositions)
			} else if len(openPositions) >= cfg.MaxOpenPositions {
				t.logInfo("Maximum open positions reached, waiting to close positions", "current", len(openPositions), "max", cfg.MaxOpenPositions)
			}
		}

		// Sleep before next iteration
		time.Sleep(time.Duration(cfg.LoopDelaySeconds) * time.Second)
	}
}

// runHealthCheck logs a status summary and optionally sends it to Slack.
func (t *EVMTrader) runHealthCheck(ctx context.Context, cfg AutoTradingConfig, currentBlock uint64, openPositions []ParsedTrade, marketIndices, collateralIndices []uint64) {
	fields := map[string]any{
		"chain_id":        t.cfg.ChainID,
		"account":         t.ethAddrBech32,
		"current_block":   currentBlock,
		"open_positions":  len(openPositions),
		"max_positions":   cfg.MaxOpenPositions,
		"market_indices":  formatUint64Slice(marketIndices),
		"collateral_idxs": formatUint64Slice(collateralIndices),
	}
	for _, collateralIdx := range collateralIndices {
		denom := t.GetCollateralDenom(collateralIdx)
		balance, err := t.queryCosmosBalance(ctx, t.ethAddrBech32, denom)
		if err != nil {
			fields["balance_"+denom] = "query_error"
			continue
		}
		fields["balance_"+denom] = balance.String()
	}
	t.logInfo("Health check", flattenHealthFields(fields)...)
	if t.cfg.SlackWebhook != "" {
		sendSlackHealthNotification(t.cfg.SlackWebhook, fields)
	}
}

// flattenHealthFields converts a map to key, value, key, value for logInfo.
func flattenHealthFields(m map[string]any) []any {
	out := make([]any, 0, len(m)*2)
	for k, v := range m {
		out = append(out, k, v)
	}
	return out
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

func extractSymbolFromDenom(denom string) string {
	if idx := strings.LastIndex(denom, "/"); idx >= 0 {
		return denom[idx+1:]
	}

	return denom
}

func formatUint64Slice(slice []uint64) string {
	if len(slice) == 0 {
		return "[]"
	}
	strs := make([]string, len(slice))
	for i, v := range slice {
		strs[i] = fmt.Sprintf("%d", v)
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

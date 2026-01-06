package evmtrader

import (
	"context"
	"fmt"
	"math/big"
)

const (
	// Default slippage percentage
	defaultSlippagePercent = "1"
)

// prepareTradeFromConfig prepares trade parameters from the trader's config.
// This is the main orchestration function that coordinates the trade preparation process.
func (t *EVMTrader) prepareTradeFromConfig(ctx context.Context, balance *big.Int) (*OpenTradeParams, error) {
	// Step 1: Determine collateral index
	// If user provided a collateral index, use it. Otherwise, default to the market's quote token index.
	collateralIndex := t.cfg.CollateralIndex
	if collateralIndex == 0 {
		market, err := t.queryMarket(ctx, t.cfg.MarketIndex)
		if err != nil {
			return nil, fmt.Errorf("query market for collateral index (market=%d): %w", t.cfg.MarketIndex, err)
		}
		if market.QuoteToken == nil {
			return nil, fmt.Errorf("market %d has no quote token to use as default collateral index", t.cfg.MarketIndex)
		}
		collateralIndex = *market.QuoteToken
		t.logDebug("Using quote token index as default collateral index", "market_index", t.cfg.MarketIndex, "collateral_index", collateralIndex)
	} else {
		t.logDebug("Using user-provided collateral index", "market_index", t.cfg.MarketIndex, "collateral_index", collateralIndex)
	}

	// Step 2: Determine trade amount (validates balance internally)
	// TODO: validate balance after
	tradeAmt := new(big.Int).SetUint64(t.cfg.TradeSize)

	// Step 3: Determine trade parameters (pure logic, no I/O)
	leverage := t.determineLeverage()
	isLong := t.determineDirection()
	tradeType := t.determineTradeType()

	// Step 4: Determine open_price
	// - If set in config (from CLI --open-price flag), use that
	// - Otherwise, fetch from oracle
	var openPrice float64
	var userProvidedPrice bool
	if t.cfg.OpenPrice != nil {
		openPrice = *t.cfg.OpenPrice
		userProvidedPrice = true
		t.logDebug("Using open_price from config", "price", openPrice)
	} else {
		// Fetch market price from oracle (I/O operation)
		price, err := t.fetchMarketPrice(ctx, tradeType)
		if err != nil {
			return nil, err
		}
		if price == 0 {
			t.logWarn("Oracle price is zero, skipping trade")
			return nil, nil
		}
		openPrice = price
		userProvidedPrice = false
		t.logDebug("Fetched open_price from oracle", "price", openPrice)
	}

	// Step 5: Adjust price for limit orders
	// Only adjust if price was fetched from oracle, not user-provided
	isLimitOrder := isLimitOrStopOrder(tradeType)
	adjustedPrice := t.adjustPriceForLimitOrder(openPrice, isLong, isLimitOrder, userProvidedPrice)

	// Validate adjusted price is non-zero for limit/stop orders (required by specification)
	if isLimitOrder && adjustedPrice == 0 {
		return nil, fmt.Errorf("adjusted open_price cannot be zero for %s orders (trigger price required)", tradeType)
	}

	// Step 6: Log and return
	t.logTradePreparation(tradeType, isLong, leverage, tradeAmt, adjustedPrice, openPrice, nil, nil)

	return &OpenTradeParams{
		MarketIndex:     t.cfg.MarketIndex,
		Leverage:        leverage,
		Long:            isLong,
		CollateralIndex: collateralIndex,
		TradeType:       tradeType,
		OpenPrice:       &adjustedPrice,
		TP:              nil, // Only set if explicitly provided
		SL:              nil, // Only set if explicitly provided
		SlippageP:       defaultSlippagePercent,
		CollateralAmt:   tradeAmt,
	}, nil
}

// determineTradeAmount calculates the trade amount based on user-provided config only.
// Returns nil if balance is insufficient or no trade size is configured (not an error condition).
func (t *EVMTrader) determineTradeAmount(balance *big.Int) (*big.Int, error) {
	var tradeAmt *big.Int

	if t.cfg.TradeSize > 0 {
		// Use exact trade size from config
		tradeAmt = new(big.Int).SetUint64(t.cfg.TradeSize)
		if balance.Cmp(tradeAmt) < 0 {
			t.logWarn("Insufficient balance for trade", "balance", balance.String(), "required", tradeAmt.String())
			return nil, nil
		}
	} else {
		// Use user-provided TradeSizeMin or TradeSizeMax only (no fallback to balance)
		tradeAmt = t.calculateDeterministicTradeAmount(balance)
		if tradeAmt == nil {
			t.logWarn("Insufficient balance for trade or no trade size configured", "balance", balance.String())
			return nil, nil
		}
	}

	return tradeAmt, nil
}

// determineLeverage returns the leverage to use for the trade.
// Uses config value if set, otherwise defaults to 1.
func (t *EVMTrader) determineLeverage() uint64 {
	if t.cfg.Leverage > 0 {
		return t.cfg.Leverage
	}
	// Default to 1x leverage if not specified
	return 1
}

// determineDirection returns the trade direction (long or short).
// Uses config value if set, otherwise defaults to long (true).
func (t *EVMTrader) determineDirection() bool {
	if t.cfg.Long != nil {
		return *t.cfg.Long
	}
	// Default to long if not specified
	return true
}

// determineTradeType returns the trade type (trade, limit, or stop).
// Uses config value if set, otherwise determines based on EnableLimitOrder flag.
func (t *EVMTrader) determineTradeType() string {
	if t.cfg.TradeType != "" {
		// Use explicitly configured trade type
		return t.cfg.TradeType
	}

	// Auto-determine based on enableLimitOrder flag
	if t.cfg.EnableLimitOrder && randomBool() {
		// Randomly choose between limit and stop
		if randomBool() {
			return "stop"
		}
		return "limit"
	}

	return "trade" // Market order
}

// fetchMarketPrice queries the oracle for the appropriate price based on trade type.
// For market orders, it fetches the exchange rate between base and quote tokens.
// For limit/stop orders, it fetches the collateral token price.
func (t *EVMTrader) fetchMarketPrice(ctx context.Context, tradeType string) (float64, error) {
	if tradeType == TradeTypeMarket {
		// For market orders, get the exchange rate (base per quote)
		return t.fetchExchangeRateForMarket(ctx)
	}

	// For limit/stop orders, use collateral price
	price, err := t.queryOraclePrice(ctx, t.cfg.CollateralIndex)
	if err != nil {
		return 0, fmt.Errorf("query collateral price (index=%d): %w", t.cfg.CollateralIndex, err)
	}
	return price, nil
}

// fetchExchangeRateForMarket queries the market and returns the exchange rate between base and quote tokens.
// This matches what the perp contract does internally. Uses the market index from config.
func (t *EVMTrader) fetchExchangeRateForMarket(ctx context.Context) (float64, error) {
	return t.fetchMarketPriceForIndex(ctx, t.cfg.MarketIndex)
}

// fetchMarketPriceForIndex queries the oracle for the exchange rate of a specific market.
// This fetches the base/quote exchange rate for the given market index.
func (t *EVMTrader) fetchMarketPriceForIndex(ctx context.Context, marketIndex uint64) (float64, error) {
	// Query market to get base and quote token indices
	market, err := t.queryMarket(ctx, marketIndex)
	if err != nil {
		return 0, fmt.Errorf("query market %d: %w", marketIndex, err)
	}

	if market.BaseToken == nil {
		return 0, fmt.Errorf("market %d has no base token", marketIndex)
	}
	if market.QuoteToken == nil {
		return 0, fmt.Errorf("market %d has no quote token", marketIndex)
	}

	// Query the exchange rate
	rate, err := t.queryExchangeRate(ctx, *market.BaseToken, *market.QuoteToken)
	if err != nil {
		return 0, fmt.Errorf("query exchange rate for market %d (base=%d, quote=%d): %w",
			marketIndex, *market.BaseToken, *market.QuoteToken, err)
	}

	return rate, nil
}

// adjustPriceForLimitOrder returns the price unchanged.
// All prices (user-provided and oracle-fetched) are used as-is without any adjustment.
func (t *EVMTrader) adjustPriceForLimitOrder(price float64, isLong, isLimitOrder, userProvided bool) float64 {
	return price
}

// logTradePreparation logs the prepared trade parameters.
func (t *EVMTrader) logTradePreparation(tradeType string, isLong bool, leverage uint64,
	tradeAmt *big.Int, adjustedPrice, oraclePrice float64, tp, sl *float64) {

	whatTraderOpens := "position"
	if isLimitOrStopOrder(tradeType) {
		whatTraderOpens = "limit order"
	}

	t.logInfo("Opening trade",
		"type", whatTraderOpens,
		"long", isLong,
		"leverage", leverage,
		"collateral", tradeAmt.String(),
		"open_price", adjustedPrice,
		"oracle_price", oraclePrice,
		"tp", tp,
		"sl", sl,
	)
}

// calculateDeterministicTradeAmount calculates a deterministic trade amount based on user-provided config only.
// Uses TradeSizeMin if set, otherwise TradeSizeMax. Returns nil if balance is insufficient or no size is configured.
func (t *EVMTrader) calculateDeterministicTradeAmount(balance *big.Int) *big.Int {
	// Prefer TradeSizeMin if set
	if t.cfg.TradeSizeMin > 0 {
		amt := new(big.Int).SetUint64(t.cfg.TradeSizeMin)
		if balance.Cmp(amt) < 0 {
			// Insufficient balance - return nil (don't use available balance)
			return nil
		}
		// If TradeSizeMax is set and larger than min, use TradeSizeMax (if balance allows)
		if t.cfg.TradeSizeMax > t.cfg.TradeSizeMin {
			maxAmt := new(big.Int).SetUint64(t.cfg.TradeSizeMax)
			if balance.Cmp(maxAmt) < 0 {
				// Insufficient balance for max - return nil (don't use available balance)
				return nil
			}
			// Use max if balance is sufficient
			return maxAmt
		}
		return amt
	}

	// If TradeSizeMin not set, try TradeSizeMax
	if t.cfg.TradeSizeMax > 0 {
		amt := new(big.Int).SetUint64(t.cfg.TradeSizeMax)
		if balance.Cmp(amt) < 0 {
			// Insufficient balance - return nil (don't use available balance)
			return nil
		}
		return amt
	}

	// If neither min nor max is set, return nil (no user input)
	return nil
}

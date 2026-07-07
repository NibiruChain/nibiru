package evmtrader

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	// Price adjustment percentages for limit orders
	limitOrderPriceAdjustmentUp   = 1.1 // 10% above for long positions
	limitOrderPriceAdjustmentDown = 0.9 // 10% below for short positions

	// Default slippage percentage
	defaultSlippagePercent = "1"
)

// prepareTradeFromConfig prepares trade parameters from the trader's config.
// This is the main orchestration function that coordinates the trade preparation process.
func (t *EVMTrader) prepareTradeFromConfig(ctx context.Context, balance *big.Int) (*OpenTradeParams, error) {
	// Step 1: Determine trade amount (validates balance internally)
	tradeAmt, err := t.determineTradeAmount(balance)
	if err != nil {
		return nil, err
	}
	if tradeAmt == nil {
		// Insufficient balance - not an error, just skip this trade
		return nil, nil
	}

	// Step 2: Determine trade parameters (pure logic, no I/O)
	leverage := t.determineLeverage()
	isLong := t.determineDirection()
	tradeType := t.determineTradeType()

	// Step 3: Determine open_price
	// - If set in config (from CLI --open-price flag), use that
	// - Otherwise, fetch from oracle
	var openPrice float64
	if t.cfg.OpenPrice != nil {
		openPrice = *t.cfg.OpenPrice
		t.log("Using open_price from config", "price", openPrice)
	} else {
		// Fetch market price from oracle (I/O operation)
		price, err := t.fetchMarketPrice(ctx, tradeType)
		if err != nil {
			return nil, err
		}
		if price == 0 {
			t.log("Oracle price is zero, skipping trade")
			return nil, nil
		}
		openPrice = price
		t.log("Fetched open_price from oracle", "price", openPrice)
	}

	// Step 4: Adjust price for limit orders
	isLimitOrder := (tradeType == "limit" || tradeType == "stop")
	adjustedPrice := t.adjustPriceForLimitOrder(openPrice, isLong, isLimitOrder)

	// Validate adjusted price is non-zero for limit/stop orders (required by specification)
	if isLimitOrder && adjustedPrice == 0 {
		return nil, fmt.Errorf("adjusted open_price cannot be zero for %s orders (trigger price required)", tradeType)
	}

	// Step 5: Calculate TP/SL for limit orders
	var tp, sl *float64
	if isLimitOrder {
		tpVal, slVal := t.calculateTPSL(adjustedPrice, isLong)
		tp = &tpVal
		sl = &slVal
	}

	// Step 6: Log and return
	t.logTradePreparation(tradeType, isLong, leverage, tradeAmt, adjustedPrice, openPrice, tp, sl)

	return &OpenTradeParams{
		MarketIndex:     t.cfg.MarketIndex,
		Leverage:        leverage,
		Long:            isLong,
		CollateralIndex: t.cfg.CollateralIndex,
		TradeType:       tradeType,
		OpenPrice:       &adjustedPrice,
		TP:              tp,
		SL:              sl,
		SlippageP:       defaultSlippagePercent,
		CollateralAmt:   tradeAmt,
	}, nil
}

// determineTradeAmount calculates the trade amount based on config and available balance.
// Returns nil if balance is insufficient (not an error condition).
func (t *EVMTrader) determineTradeAmount(balance *big.Int) (*big.Int, error) {
	var tradeAmt *big.Int

	if t.cfg.TradeSize > 0 {
		// Use exact trade size from config
		tradeAmt = new(big.Int).SetUint64(t.cfg.TradeSize)
		if balance.Cmp(tradeAmt) < 0 {
			t.log("Insufficient ERC20 balance for trade", "balance", balance.String(), "required", tradeAmt.String())
			return nil, nil
		}
	} else {
		// Use random amount within configured range
		tradeAmt = t.calculateRandomTradeAmount(balance)
		if tradeAmt == nil {
			t.log("Insufficient ERC20 balance for trade", "balance", balance.String())
			return nil, nil
		}
	}

	return tradeAmt, nil
}

// determineLeverage returns the leverage to use for the trade.
// Uses config value if set, otherwise random within configured range.
// Ensures leverage > 0 as required by specification.
func (t *EVMTrader) determineLeverage() uint64 {
	if t.cfg.Leverage > 0 {
		return t.cfg.Leverage
	}
	leverage := t.calculateRandomLeverage()
	// Ensure leverage is at least 1 (required by specification)
	if leverage == 0 {
		return 1
	}
	return leverage
}

// determineDirection returns the trade direction (long or short).
// Uses config value if set, otherwise random using cryptographically secure randomness.
func (t *EVMTrader) determineDirection() bool {
	if t.cfg.Long != nil {
		return *t.cfg.Long
	}
	// Use crypto/rand for unpredictable trade direction
	return secureRandomBool()
}

// determineTradeType returns the trade type (trade, limit, or stop).
// Uses config value if set, otherwise determines based on EnableLimitOrder flag.
func (t *EVMTrader) determineTradeType() string {
	if t.cfg.TradeType != "" {
		// Use explicitly configured trade type
		return t.cfg.TradeType
	}

	// Auto-determine based on enableLimitOrder flag
	if t.cfg.EnableLimitOrder && secureRandomBool() {
		// Randomly choose between limit and stop
		if secureRandomBool() {
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
	if tradeType == "trade" {
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

// adjustPriceForLimitOrder adjusts the price for limit orders.
// For long positions, increases price by 10% (buy limit above current price).
// For short positions, decreases price by 10% (sell limit below current price).
// For market orders, returns the price unchanged.
func (t *EVMTrader) adjustPriceForLimitOrder(price float64, isLong, isLimitOrder bool) float64 {
	if !isLimitOrder {
		return price
	}

	if isLong {
		return price * limitOrderPriceAdjustmentUp
	}
	return price * limitOrderPriceAdjustmentDown
}

// logTradePreparation logs the prepared trade parameters.
func (t *EVMTrader) logTradePreparation(tradeType string, isLong bool, leverage uint64,
	tradeAmt *big.Int, adjustedPrice, oraclePrice float64, tp, sl *float64) {

	whatTraderOpens := "position"
	if tradeType == "limit" || tradeType == "stop" {
		whatTraderOpens = "limit order"
	}

	t.log("Opening trade",
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

// secureRandomBool returns a cryptographically secure random boolean.
// Uses crypto/rand instead of math/rand for unpredictable randomness.
func secureRandomBool() bool {
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to false on error (should never happen)
		return false
	}
	return b[0]&1 == 1
}

// calculateRandomTradeAmount calculates a random trade amount within configured range.
// Uses cryptographically secure randomness for unpredictable trade amounts.
func (t *EVMTrader) calculateRandomTradeAmount(balance *big.Int) *big.Int {
	if t.cfg.TradeSizeMax <= t.cfg.TradeSizeMin {
		// Use min if max not set
		amt := new(big.Int).SetUint64(t.cfg.TradeSizeMin)
		if balance.Cmp(amt) < 0 {
			return nil
		}
		return amt
	}

	// Random amount between min and max using crypto/rand
	rangeSize := t.cfg.TradeSizeMax - t.cfg.TradeSizeMin
	randomOffset := secureRandomUint64(rangeSize + 1)
	tradeAmt := t.cfg.TradeSizeMin + randomOffset

	amt := new(big.Int).SetUint64(tradeAmt)
	if balance.Cmp(amt) < 0 {
		// If balance is less than min, try to use what we have
		if balance.Cmp(big.NewInt(0)) > 0 {
			return balance
		}
		return nil
	}
	return amt
}

// calculateRandomLeverage calculates a random leverage within configured range.
// Uses cryptographically secure randomness for unpredictable leverage selection.
func (t *EVMTrader) calculateRandomLeverage() uint64 {
	if t.cfg.LeverageMax <= t.cfg.LeverageMin {
		return t.cfg.LeverageMin
	}
	rangeSize := t.cfg.LeverageMax - t.cfg.LeverageMin
	randomOffset := secureRandomUint64(rangeSize + 1)
	return t.cfg.LeverageMin + randomOffset
}

// secureRandomUint64 returns a cryptographically secure random uint64 in the range [0, max).
func secureRandomUint64(max uint64) uint64 {
	if max == 0 {
		return 0
	}

	// Generate random big.Int
	maxBig := new(big.Int).SetUint64(max)
	n, err := rand.Int(rand.Reader, maxBig)
	if err != nil {
		// Fallback to 0 on error (should never happen)
		return 0
	}
	return n.Uint64()
}

// calculateTPSL calculates take profit and stop loss based on open price and direction
func (t *EVMTrader) calculateTPSL(openPrice float64, isLong bool) (tp, sl float64) {
	if isLong {
		// Long: TP above, SL below
		tp = openPrice * 1.5
		sl = openPrice / 1.5
	} else {
		// Short: TP below, SL above
		tp = openPrice / 1.5
		sl = openPrice * 1.5
	}
	return tp, sl
}

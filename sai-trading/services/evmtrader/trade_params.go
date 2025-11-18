package evmtrader

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
)

// prepareTradeFromConfig prepares trade parameters from the trader's config
func (t *EVMTrader) prepareTradeFromConfig(ctx context.Context, balance *big.Int) (*OpenTradeParams, error) {
	// Calculate trade amount - use exact value if provided, otherwise random within range
	var tradeAmt *big.Int
	if t.cfg.TradeSize > 0 {
		// Use exact trade size
		tradeAmt = new(big.Int).SetUint64(t.cfg.TradeSize)
		if balance.Cmp(tradeAmt) < 0 {
			t.log("Insufficient ERC20 balance for trade", "balance", balance.String(), "required", tradeAmt.String())
			return nil, nil
		}
	} else {
		// Use random within range
		tradeAmt = t.calculateRandomTradeAmount(balance)
		if tradeAmt == nil {
			t.log("Insufficient ERC20 balance for trade", "balance", balance.String())
			return nil, nil
		}
	}

	// Calculate leverage - use exact value if provided, otherwise random within range
	var leverage uint64
	if t.cfg.Leverage > 0 {
		leverage = t.cfg.Leverage
	} else {
		leverage = t.calculateRandomLeverage()
	}

	// Determine trade direction - use config if provided, otherwise random
	var isLong bool
	if t.cfg.Long != nil {
		isLong = *t.cfg.Long
	} else {
		isLong = rand.Intn(2) == 1
	}

	// Determine trade type - use config if set, otherwise auto-determine
	var tradeType string
	var isLimitOrder bool
	if t.cfg.TradeType != "" {
		// Use explicitly set trade type
		tradeType = t.cfg.TradeType
		isLimitOrder = (tradeType == "limit" || tradeType == "stop")
	} else {
		// Auto-determine based on enableLimitOrder flag
		isLimitOrder = t.cfg.EnableLimitOrder && rand.Intn(2) == 1
		if isLimitOrder {
			if rand.Intn(2) == 1 {
				tradeType = "stop"
			} else {
				tradeType = "limit"
			}
		} else {
			tradeType = "trade"
		}
	}

	// Fetch current price from oracle
	// For market orders, we need the market's base token price, not the collateral price
	var openPrice float64
	var err error
	if tradeType == "trade" {
		// For market orders, query the market to get base and quote tokens, then get the exchange rate
		// This matches what the contract does - it queries GetExchangeRate to get base_per_quote
		market, err := t.queryMarket(ctx, t.cfg.MarketIndex)
		if err != nil {
			return nil, fmt.Errorf("query market %d: %w", t.cfg.MarketIndex, err)
		}

		if market.BaseToken == nil {
			return nil, fmt.Errorf("market %d has no base token", t.cfg.MarketIndex)
		}
		if market.QuoteToken == nil {
			return nil, fmt.Errorf("market %d has no quote token", t.cfg.MarketIndex)
		}

		// Query the exchange rate between base and quote tokens (this is what the contract uses)
		openPrice, err = t.queryExchangeRate(ctx, *market.BaseToken, *market.QuoteToken)
		if err != nil {
			return nil, fmt.Errorf("query exchange rate for market %d (base=%d, quote=%d): %w", t.cfg.MarketIndex, *market.BaseToken, *market.QuoteToken, err)
		}
	} else {
		// For limit/stop orders, use collateral price
		openPrice, err = t.queryOraclePrice(ctx, t.cfg.CollateralIndex)
		if err != nil {
			return nil, fmt.Errorf("query collateral price (index=%d): %w", t.cfg.CollateralIndex, err)
		}
	}
	if openPrice == 0 {
		t.log("Oracle price is zero, skipping trade")
		return nil, nil
	}

	// Adjust price for limit orders
	if isLimitOrder {
		if isLong {
			openPrice = openPrice * 1.1 // Buy limit above current price
		} else {
			openPrice = openPrice * 0.9 // Sell limit below current price
		}
	}

	// For market trades, the contract uses the actual execution price from the market
	// However, open_price is still required by the contract for liquidation price calculation.
	// We use the oracle price as an estimate - the contract will validate against the actual
	// execution price during trade validation.
	// For limit/stop orders, open_price is the limit/stop price.
	marketOpenPrice := openPrice

	// Calculate TP/SL
	var tp, sl *float64
	if isLimitOrder {
		tpVal, slVal := t.calculateTPSL(openPrice, isLong)
		tp = &tpVal
		sl = &slVal
	}

	whatTraderOpens := "position"
	if isLimitOrder {
		whatTraderOpens = "limit order"
	}

	t.log("Opening trade",
		"type", whatTraderOpens,
		"long", isLong,
		"leverage", leverage,
		"collateral", tradeAmt.String(),
		"open_price", marketOpenPrice,
		"oracle_price", openPrice,
		"tp", tp,
		"sl", sl,
	)

	return &OpenTradeParams{
		MarketIndex:     t.cfg.MarketIndex,
		Leverage:        leverage,
		Long:            isLong,
		CollateralIndex: t.cfg.CollateralIndex,
		TradeType:       tradeType,
		OpenPrice:       &marketOpenPrice,
		TP:              tp,
		SL:              sl,
		SlippageP:       "1", // TODO: Make this configurable
		CollateralAmt:   tradeAmt,
	}, nil
}

// calculateRandomTradeAmount calculates a random trade amount within configured range
func (t *EVMTrader) calculateRandomTradeAmount(balance *big.Int) *big.Int {
	if t.cfg.TradeSizeMax <= t.cfg.TradeSizeMin {
		// Use min if max not set
		amt := new(big.Int).SetUint64(t.cfg.TradeSizeMin)
		if balance.Cmp(amt) < 0 {
			return nil
		}
		return amt
	}

	// Random amount between min and max
	rangeSize := t.cfg.TradeSizeMax - t.cfg.TradeSizeMin
	randomOffset := rand.Uint64() % (rangeSize + 1)
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

// calculateRandomLeverage calculates a random leverage within configured range
func (t *EVMTrader) calculateRandomLeverage() uint64 {
	if t.cfg.LeverageMax <= t.cfg.LeverageMin {
		return t.cfg.LeverageMin
	}
	rangeSize := t.cfg.LeverageMax - t.cfg.LeverageMin
	randomOffset := rand.Uint64() % (rangeSize + 1)
	return t.cfg.LeverageMin + randomOffset
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

package evmtrader

import (
	"context"
	"math/big"
	"math/rand"
)

// prepareTradeFromConfig prepares trade parameters from the trader's config
func (t *EVMTrader) prepareTradeFromConfig(ctx context.Context, balance *big.Int) (*OpenTradeParams, error) {
	// Calculate random trade amount and leverage
	tradeAmt := t.calculateRandomTradeAmount(balance)
	if tradeAmt == nil {
		t.log("Insufficient ERC20 balance for trade", "balance", balance.String())
		return nil, nil
	}

	leverage := t.calculateRandomLeverage()
	isLong := rand.Intn(2) == 1
	isLimitOrder := t.cfg.EnableLimitOrder && rand.Intn(2) == 1

	// Fetch current price from oracle
	openPrice, err := t.queryOraclePrice(ctx, t.cfg.CollateralIndex)
	if err != nil {
		t.log("Failed to fetch oracle price", "error", err.Error())
		return nil, err
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

	// Determine trade type
	tradeType := "trade"
	if isLimitOrder {
		if rand.Intn(2) == 1 {
			tradeType = "stop"
		} else {
			tradeType = "limit"
		}
	}

	// For market trades, the contract uses the actual execution price from the market
	// The open_price parameter is used for liquidation price calculation, so we use
	// the oracle price as an estimate. The contract will validate against the actual
	// execution price during trade validation.
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
		OpenPrice:       marketOpenPrice,
		TP:              tp,
		SL:              sl,
		SlippageP:       "1", // 100% slippage tolerance (maximum allowed)
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

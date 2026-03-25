package evmtrader

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// logTradeLifecycleCSV writes a single row per trade lifecycle (open+close) to logs/trades.csv.
// It is called when a trade closes, using the stored open metadata (lc) plus the final close trade.
func (t *EVMTrader) logTradeLifecycleCSV(tradeID uint64, lc *tradeLifecycle, closeTrade ParsedTrade, txHash string, closeHeight int64) {
	logDir := "logs"
	logPath := filepath.Join(logDir, "trades.csv")

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.logWarn("Failed to create logs directory", "path", logDir, "error", err.Error())
		return
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.logWarn("Failed to open trade lifecycle CSV", "path", logPath, "error", err.Error())
		return
	}
	defer f.Close()

	// If file was just created and is empty, write header.
	if fi, err := f.Stat(); err == nil && fi.Size() == 0 {
		w := csv.NewWriter(f)
		_ = w.Write([]string{
			"trade_id",
			"market_index",
			"direction",
			"trade_type",
			"collateral_index",
			"collateral_amount",
			"collateral_denom",
			"leverage",
			"open_time_utc",
			"open_block_height",
			"open_price",
			"close_time_utc",
			"close_block_height",
			"close_price",
			"tp",
			"sl",
		})
		w.Flush()
	}

	w := csv.NewWriter(f)
	defer w.Flush()

	// Defaults from open lifecycle metadata, if present.
	var (
		openTime   string
		openBlock  string
		openPrice  string
		marketIdx  string
		direction  string
		tradeType  string
		collIdx    string
		collAmt    string
		collDenom  string
		leverage   string
		tp         string
		sl         string
		closePrice string
	)

	if lc != nil {
		openTime = lc.OpenTimeUTC.Format(time.RFC3339)
		openBlock = strconv.FormatInt(lc.OpenBlock, 10)
		openPrice = lc.OpenPrice
		marketIdx = strconv.FormatUint(lc.MarketIndex, 10)
		direction = lc.Direction
		tradeType = lc.TradeType
		collIdx = strconv.FormatUint(lc.CollateralIndex, 10)
		collAmt = lc.CollateralAmount
		collDenom = lc.CollateralDenom
		leverage = lc.Leverage
		tp = lc.TP
		sl = lc.SL
	}

	// Overlay with close trade info where relevant / available.
	if closeTrade.MarketIndex != 0 {
		marketIdx = strconv.FormatUint(closeTrade.MarketIndex, 10)
	}
	if closeTrade.TradeType != "" {
		tradeType = closeTrade.TradeType
	}
	if closeTrade.Leverage != "" {
		leverage = closeTrade.Leverage
	}
	if closeTrade.CollateralIndex != 0 {
		collIdx = strconv.FormatUint(closeTrade.CollateralIndex, 10)
	}
	if closeTrade.CollateralAmount != "" {
		collAmt = closeTrade.CollateralAmount
	}
	if cd := t.GetCollateralDenom(closeTrade.CollateralIndex); cd != "" {
		collDenom = cd
	}
	// Direction from close trade if missing.
	if direction == "" {
		if closeTrade.Long {
			direction = "LONG"
		} else {
			direction = "SHORT"
		}
	}
	// TP/SL from close trade if present.
	if closeTrade.TP != "" {
		tp = closeTrade.TP
	}
	if closeTrade.SL != nil && *closeTrade.SL != "" {
		sl = *closeTrade.SL
	}
	// We don't have a dedicated close price field yet, but if OpenPrice is updated
 	// on close we can reuse it. For now, prefer closeTrade.OpenPrice if non-empty.
	if closeTrade.OpenPrice != "" {
		closePrice = closeTrade.OpenPrice
	}

	closeTime := time.Now().UTC().Format(time.RFC3339)
	closeBlock := strconv.FormatInt(closeHeight, 10)

	record := []string{
		strconv.FormatUint(tradeID, 10),
		marketIdx,
		direction,
		tradeType,
		collIdx,
		collAmt,
		collDenom,
		leverage,
		openTime,
		openBlock,
		openPrice,
		closeTime,
		closeBlock,
		closePrice,
		tp,
		sl,
	}

	if err := w.Write(record); err != nil {
		t.logWarn("Failed to write trade lifecycle CSV record", "path", logPath, "error", err.Error())
	}
}

// logTransactionCSV appends a single on-chain EVM transaction row to logs/transactions.csv.
// It is best-effort and should never cause trading to fail.
func (t *EVMTrader) logTransactionCSV(txHash string, status string, reason string, recipient string, height int64, gasWanted, gasUsed uint64) {
	logDir := "logs"
	logPath := filepath.Join(logDir, "transactions.csv")

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.logWarn("Failed to create logs directory for transactions", "path", logDir, "error", err.Error())
		return
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.logWarn("Failed to open transactions CSV", "path", logPath, "error", err.Error())
		return
	}
	defer f.Close()

	if fi, err := f.Stat(); err == nil && fi.Size() == 0 {
		w := csv.NewWriter(f)
		_ = w.Write([]string{
			"timestamp_utc",
			"tx_hash",
			"status",
			"reason",
			"chain_id",
			"block_height",
			"gas_wanted",
			"gas_used",
			"recipient",
		})
		w.Flush()
	}

	w := csv.NewWriter(f)
	defer w.Flush()

	ts := time.Now().UTC().Format(time.RFC3339)

	record := []string{
		ts,
		txHash,
		status,
		reason,
		t.cfg.ChainID,
		strconv.FormatInt(height, 10),
		strconv.FormatUint(gasWanted, 10),
		strconv.FormatUint(gasUsed, 10),
		recipient,
	}

	if err := w.Write(record); err != nil {
		t.logWarn("Failed to write transactions CSV record", "path", logPath, "error", err.Error())
	}
}



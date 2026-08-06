package evmtrader

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const failedTxWindow = 24 * time.Hour

type failedTxEvent struct {
	ts     time.Time
	reason string
}

// logTxResult emits a structured tx_result log and records failures for Slack health.
func (t *EVMTrader) logTxResult(txHash string, status string, reason string, recipient string, height int64, gasWanted, gasUsed uint64) {
	kv := []any{
		"tx_hash", txHash,
		"status", status,
		"recipient", recipient,
		"height", height,
		"gas_wanted", gasWanted,
		"gas_used", gasUsed,
		"chain_id", t.cfg.ChainID,
	}
	if reason != "" {
		kv = append(kv, "reason", reason)
	}
	if status == "failed" {
		t.recordFailedTx(reason)
		t.logWarn("tx_result", kv...)
		return
	}
	t.logInfo("tx_result", kv...)
}

// logTradeLifecycle emits a structured trade_lifecycle log when a trade closes.
func (t *EVMTrader) logTradeLifecycle(tradeID uint64, lc *tradeLifecycle, closeTrade ParsedTrade, closeHeight int64, closePrice string) {
	var (
		openTime  string
		openBlock int64
		openPrice string
		marketIdx uint64
		direction string
		tradeType string
		collIdx   uint64
		collAmt   string
		collDenom string
		leverage  string
		tp        string
		sl        string
	)

	if lc != nil {
		openTime = lc.OpenTimeUTC.Format(time.RFC3339)
		openBlock = lc.OpenBlock
		openPrice = lc.OpenPrice
		marketIdx = lc.MarketIndex
		direction = lc.Direction
		tradeType = lc.TradeType
		collIdx = lc.CollateralIndex
		collAmt = lc.CollateralAmount
		collDenom = lc.CollateralDenom
		leverage = lc.Leverage
		tp = lc.TP
		sl = lc.SL
	}

	if closeTrade.MarketIndex != 0 {
		marketIdx = closeTrade.MarketIndex
	}
	if closeTrade.TradeType != "" {
		tradeType = closeTrade.TradeType
	}
	if closeTrade.Leverage != "" {
		leverage = closeTrade.Leverage
	}
	if closeTrade.CollateralIndex != 0 {
		collIdx = closeTrade.CollateralIndex
	}
	if closeTrade.CollateralAmount != "" {
		collAmt = closeTrade.CollateralAmount
	}
	if cd := t.GetCollateralDenom(closeTrade.CollateralIndex); cd != "" {
		collDenom = cd
	}
	if direction == "" {
		direction = boolToDirection(closeTrade.Long)
	}
	if closeTrade.TP != "" {
		tp = closeTrade.TP
	}
	if closeTrade.SL != nil && *closeTrade.SL != "" {
		sl = *closeTrade.SL
	}
	if openPrice == "" && closeTrade.OpenPrice != "" {
		openPrice = closeTrade.OpenPrice
	}

	t.logInfo("trade_lifecycle",
		"trade_id", tradeID,
		"market_index", marketIdx,
		"direction", direction,
		"trade_type", tradeType,
		"collateral_index", collIdx,
		"collateral_amount", collAmt,
		"collateral_denom", collDenom,
		"leverage", leverage,
		"open_time_utc", openTime,
		"open_block_height", openBlock,
		"open_price", openPrice,
		"close_time_utc", time.Now().UTC().Format(time.RFC3339),
		"close_block_height", closeHeight,
		"close_price", closePrice,
		"tp", tp,
		"sl", sl,
	)
}

func (t *EVMTrader) recordFailedTx(reason string) {
	reason = normalizeFailedReason(reason)
	now := time.Now().UTC()
	t.failedTxMu.Lock()
	defer t.failedTxMu.Unlock()
	t.failedTxs = append(t.failedTxs, failedTxEvent{ts: now, reason: reason})
	t.pruneFailedTxsLocked(now)
}

func (t *EVMTrader) pruneFailedTxsLocked(now time.Time) {
	cutoff := now.Add(-failedTxWindow)
	i := 0
	for i < len(t.failedTxs) && t.failedTxs[i].ts.Before(cutoff) {
		i++
	}
	if i > 0 {
		t.failedTxs = append([]failedTxEvent(nil), t.failedTxs[i:]...)
	}
}

// failedTxs24h returns count and a Slack-friendly reason summary for the last 24h.
func (t *EVMTrader) failedTxs24h(now time.Time) (count int, reasonsBlock string) {
	t.failedTxMu.Lock()
	defer t.failedTxMu.Unlock()
	t.pruneFailedTxsLocked(now.UTC())
	count = len(t.failedTxs)
	if count == 0 {
		return 0, ""
	}

	freq := map[string]int{}
	for _, ev := range t.failedTxs {
		freq[ev.reason]++
	}
	type pair struct {
		reason string
		n      int
	}
	pairs := make([]pair, 0, len(freq))
	for reason, n := range freq {
		pairs = append(pairs, pair{reason: reason, n: n})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].n != pairs[j].n {
			return pairs[i].n > pairs[j].n
		}
		return pairs[i].reason < pairs[j].reason
	})

	lines := make([]string, 0, len(pairs))
	for _, p := range pairs {
		lines = append(lines, fmt.Sprintf("• %s (%d)", p.reason, p.n))
	}
	return count, strings.Join(lines, "\n")
}

func normalizeFailedReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "unknown"
	}
	if idx := strings.Index(reason, "\n"); idx >= 0 {
		reason = strings.TrimSpace(reason[:idx])
	}
	if len(reason) > 120 {
		return reason[:117] + "..."
	}
	return reason
}

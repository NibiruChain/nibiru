package evmtrader

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type status24hMetrics struct {
	positionsOpened24h int
	positionsClosed24h int
	failedTxs24h       int
	failedReasonsBlock string
	volumeUSD          float64
	realizedPnL        float64
}

func compute24hMetrics(ctx context.Context, now time.Time, logsDir string, keeper *KeeperClient, traderAddr string) status24hMetrics {
	cutoff := now.Add(-24 * time.Hour)
	out := status24hMetrics{}

	tradesPath := filepath.Join(logsDir, "trades.csv")
	if f, err := os.Open(tradesPath); err == nil {
		defer f.Close()
		r := csv.NewReader(f)
		header, err := r.Read()
		if err == nil {
			idx := make(map[string]int, len(header))
			for i, h := range header {
				idx[h] = i
			}
			openCol, okOpen := idx["open_time_utc"]
			closeCol, okClose := idx["close_time_utc"]
			if okOpen && okClose {
				for {
					rec, err := r.Read()
					if err != nil {
						break
					}
					openT, err := time.Parse(time.RFC3339, strings.TrimSpace(rec[openCol]))
					if err == nil && !openT.Before(cutoff) {
						out.positionsOpened24h++
					}
					closeT, err := time.Parse(time.RFC3339, strings.TrimSpace(rec[closeCol]))
					if err == nil && !closeT.Before(cutoff) {
						out.positionsClosed24h++
					}
				}
			}
		}
	}

	// count failed txs in last 24h and report all unique reason kinds.
	txsPath := filepath.Join(logsDir, "transactions.csv")
	reasonCounts := map[string]int{}
	if f, err := os.Open(txsPath); err == nil {
		defer f.Close()
		r := csv.NewReader(f)
		header, err := r.Read()
		if err == nil {
			idx := make(map[string]int, len(header))
			for i, h := range header {
				idx[h] = i
			}
			tsCol, okTS := idx["timestamp_utc"]
			statusCol, okStatus := idx["status"]
			reasonCol, okReason := idx["reason"]
			if okTS && okStatus && okReason {
				for {
					rec, err := r.Read()
					if err != nil {
						break
					}
					ts, err := time.Parse(time.RFC3339, strings.TrimSpace(rec[tsCol]))
					if err != nil || ts.Before(cutoff) {
						continue
					}
					status := strings.TrimSpace(rec[statusCol])
					if status != "failed" {
						continue
					}
					out.failedTxs24h++
					reason := strings.TrimSpace(rec[reasonCol])
					if reason == "" {
						reason = "unknown"
					}
					reasonCounts[reason]++
				}
			}
		}
	}

	out.failedReasonsBlock = formatFailedReasonsBlock(reasonCounts)

	if keeper != nil && ctx != nil && ctx.Err() == nil && traderAddr != "" {
		if stats, err := keeper.ComputeUser24hStats(ctx, traderAddr); err == nil {
			out.volumeUSD = stats.VolumeUSD
			out.realizedPnL = stats.RealizedPnL
		}
	}

	return out
}

func formatFailedReasonsBlock(counts map[string]int) string {
	if len(counts) == 0 {
		return "unknown"
	}

	type kv struct {
		reason string
		count  int
	}
	all := make([]kv, 0, len(counts))
	for reason, n := range counts {
		all = append(all, kv{reason: normalizeReason(reason), count: n})
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].count != all[j].count {
			return all[i].count > all[j].count
		}
		return all[i].reason < all[j].reason
	})

	// Keep Slack message size bounded.
	const maxKinds = 25
	moreKinds := 0
	if len(all) > maxKinds {
		moreKinds = len(all) - maxKinds
		all = all[:maxKinds]
	}

	lines := make([]string, 0, len(all)+1)
	for _, item := range all {
		lines = append(lines, "- "+item.reason+" ("+strconv.Itoa(item.count)+")")
	}
	if moreKinds > 0 {
		lines = append(lines, "- ... ("+strconv.Itoa(moreKinds)+" more kinds)")
	}
	return strings.Join(lines, "\n")
}

func normalizeReason(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	if len(s) > 80 {
		return s[:77] + "..."
	}
	return s
}

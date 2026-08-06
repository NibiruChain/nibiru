package evmtrader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// DefaultKeeperGraphQLURLs by network mode.
const (
	DefaultKeeperGraphQLURLTestnet = "https://sai-keeper.testnet-2.nibiru.fi/query"
	DefaultKeeperGraphQLURLMainnet = "https://sai-keeper.nibiru.fi/query"
)

// KeeperGraphQLClient queries sai-keeper GraphQL for trader activity metrics.
type KeeperGraphQLClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewKeeperGraphQLClient creates a GraphQL client for the given endpoint.
func NewKeeperGraphQLClient(endpoint string) *KeeperGraphQLClient {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil
	}
	return &KeeperGraphQLClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DefaultKeeperGraphQLURL returns the public sai-keeper GraphQL URL for a network mode.
func DefaultKeeperGraphQLURL(networkMode string) string {
	switch strings.ToLower(strings.TrimSpace(networkMode)) {
	case "testnet":
		return DefaultKeeperGraphQLURLTestnet
	case "mainnet":
		return DefaultKeeperGraphQLURLMainnet
	default:
		return ""
	}
}

// GraphQLTraderActivity holds activity metrics derived from sai-keeper.
type GraphQLTraderActivity struct {
	PositionsOpened24h int
	PositionsClosed24h int
	VolumeUSD          float64
	RealizedPnLUSD     float64
	HasVolumePnL       bool
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type tradeHistoryPage struct {
	Perp struct {
		TradeHistory []tradeHistoryItem `json:"tradeHistory"`
	} `json:"perp"`
}

type tradeHistoryItem struct {
	TradeChangeType       string   `json:"tradeChangeType"`
	RealizedPnlCollateral *float64 `json:"realizedPnlCollateral"`
	CollateralPrice       float64  `json:"collateralPrice"`
	Block                 struct {
		BlockTS flexTime `json:"block_ts"`
	} `json:"block"`
	Trade *struct {
		CollateralAmount int64   `json:"collateralAmount"`
		Leverage         float64 `json:"leverage"`
	} `json:"trade"`
}

type portfolioStatsPage struct {
	Perp struct {
		StatsUserPortfolio []struct {
			Timestamp               flexTime `json:"timestamp"`
			VolumeUSD               float64  `json:"volumeUSD"`
			RealizedPnlUSD          float64  `json:"realizedPnlUSD"`
			VolumeUSDCumulative     float64  `json:"volumeUSDCumulative"`
			RealizedPnlUSDCumulative float64  `json:"realizedPnlUSDCumulative"`
		} `json:"statsUserPortfolio"`
	} `json:"perp"`
}

type tradeHistoryActivity struct {
	opened             int
	closed             int
	estimatedVolumeUSD float64
	realizedPnLUSD     float64
}

// flexTime unmarshals GraphQL Time scalars that may arrive as RFC3339 strings.
type flexTime struct {
	time.Time
}

func (t *flexTime) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return err
	}
	t.Time = parsed.UTC()
	return nil
}

// Compute24hActivity counts opens/closes from tradeHistory and derives volume/PnL
// from trade notionals (portfolio hourly buckets often lag or stay 0 on quiet wallets).
func (c *KeeperGraphQLClient) Compute24hActivity(ctx context.Context, trader string, now time.Time) (GraphQLTraderActivity, error) {
	var out GraphQLTraderActivity
	if c == nil || c.endpoint == "" {
		return out, fmt.Errorf("graphql client not configured")
	}
	trader = strings.TrimSpace(trader)
	if trader == "" {
		return out, fmt.Errorf("trader address required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := now.Add(-24 * time.Hour)

	hist, err := c.tradeHistoryActivity24h(ctx, trader, cutoff)
	if err != nil {
		return out, err
	}
	out.PositionsOpened24h = hist.opened
	out.PositionsClosed24h = hist.closed
	out.VolumeUSD = hist.estimatedVolumeUSD
	out.RealizedPnLUSD = hist.realizedPnLUSD
	out.HasVolumePnL = true

	// Prefer portfolio stats when they have non-zero volume (better indexed aggregate).
	if vol, pnl, ok, err := c.sumPortfolioStats24h(ctx, trader, cutoff); err == nil && ok {
		if vol > 0 {
			out.VolumeUSD = vol
		}
		// Keep history PnL when portfolio rounds/lags to ~0 but closes exist.
		if math.Abs(pnl) > math.Abs(out.RealizedPnLUSD) || out.RealizedPnLUSD == 0 {
			if pnl != 0 || hist.closed == 0 {
				out.RealizedPnLUSD = pnl
			}
		}
	}

	return out, nil
}

func (c *KeeperGraphQLClient) tradeHistoryActivity24h(ctx context.Context, trader string, cutoff time.Time) (tradeHistoryActivity, error) {
	var out tradeHistoryActivity
	const pageSize = 100
	const maxPages = 50
	const collateralDecimals = 1e6

	for page := 0; page < maxPages; page++ {
		offset := page * pageSize
		query := fmt.Sprintf(`{
  perp {
    tradeHistory(where: { trader: %q }, limit: %d, offset: %d, order_desc: true, order_by: sequence) {
      tradeChangeType
      realizedPnlCollateral
      collateralPrice
      block { block_ts }
      trade { collateralAmount leverage }
    }
  }
}`, trader, pageSize, offset)

		var pageData tradeHistoryPage
		if err := c.doQuery(ctx, query, nil, &pageData); err != nil {
			return out, err
		}
		items := pageData.Perp.TradeHistory
		if len(items) == 0 {
			break
		}

		newest := time.Time{}
		inWindow := 0
		for _, item := range items {
			ts := item.Block.BlockTS.Time
			if ts.IsZero() {
				continue
			}
			if ts.After(newest) {
				newest = ts
			}
			if ts.Before(cutoff) {
				continue
			}
			inWindow++

			switch item.TradeChangeType {
			case "position_opened":
				out.opened++
				out.estimatedVolumeUSD += estimateNotionalUSD(item, collateralDecimals)
			case "position_closed_user", "position_closed_tp", "position_closed_sl", "position_liquidated":
				out.closed++
				// Count close notional too (round-trip volume).
				out.estimatedVolumeUSD += estimateNotionalUSD(item, collateralDecimals)
				if item.RealizedPnlCollateral != nil {
					out.realizedPnLUSD += (*item.RealizedPnlCollateral / collateralDecimals) * item.CollateralPrice
				}
			}
		}
		if inWindow == 0 && (newest.IsZero() || !newest.After(cutoff)) {
			break
		}
		if len(items) < pageSize {
			break
		}
	}
	return out, nil
}

func estimateNotionalUSD(item tradeHistoryItem, collateralDecimals float64) float64 {
	if item.Trade == nil || item.CollateralPrice == 0 {
		return 0
	}
	collateral := float64(item.Trade.CollateralAmount) / collateralDecimals
	lev := item.Trade.Leverage
	if lev <= 0 {
		lev = 1
	}
	return collateral * lev * item.CollateralPrice
}

func (c *KeeperGraphQLClient) sumPortfolioStats24h(ctx context.Context, trader string, cutoff time.Time) (volumeUSD, realizedPnL float64, ok bool, err error) {
	// range "7d" returns hourly points on sai-keeper; use cumulative delta when possible.
	query := fmt.Sprintf(`{
  perp {
    statsUserPortfolio(trader: %q, range: "7d") {
      timestamp
      volumeUSD
      realizedPnlUSD
      volumeUSDCumulative
      realizedPnlUSDCumulative
    }
  }
}`, trader)

	var pageData portfolioStatsPage
	if err := c.doQuery(ctx, query, nil, &pageData); err != nil {
		return 0, 0, false, err
	}
	rows := pageData.Perp.StatsUserPortfolio
	if len(rows) == 0 {
		return 0, 0, false, nil
	}

	var (
		sumVol, sumPnl float64
		matched        bool
		beforeIdx      = -1
	)
	for i, row := range rows {
		if !row.Timestamp.Time.After(cutoff) {
			beforeIdx = i
		}
		if row.Timestamp.Time.Before(cutoff) {
			continue
		}
		sumVol += row.VolumeUSD
		sumPnl += row.RealizedPnlUSD
		matched = true
	}

	volumeUSD = sumVol
	realizedPnL = sumPnl
	if beforeIdx >= 0 {
		before := rows[beforeIdx]
		latest := rows[len(rows)-1]
		cumVol := latest.VolumeUSDCumulative - before.VolumeUSDCumulative
		cumPnl := latest.RealizedPnlUSDCumulative - before.RealizedPnlUSDCumulative
		if cumVol > volumeUSD {
			volumeUSD = cumVol
		}
		if math.Abs(cumPnl) > math.Abs(realizedPnL) {
			realizedPnL = cumPnl
		}
		matched = true
	}

	return volumeUSD, realizedPnL, matched, nil
}

func (c *KeeperGraphQLClient) doQuery(ctx context.Context, query string, variables map[string]any, dest any) error {
	body, err := json.Marshal(gqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("read graphql response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("graphql http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var gqlResp gqlResponse
	if err := json.Unmarshal(raw, &gqlResp); err != nil {
		return fmt.Errorf("unmarshal graphql response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(gqlResp.Data, dest); err != nil {
		return fmt.Errorf("unmarshal graphql data: %w", err)
	}
	return nil
}

package evmtrader

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type KeeperClient struct {
	db *sql.DB
}

type KeeperUser24hStats struct {
	VolumeUSD   float64
	RealizedPnL float64
}

func NewKeeperClient(dsn string) (*KeeperClient, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open keeper db: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(30 * time.Minute)

	return &KeeperClient{db: db}, nil
}

func (kc *KeeperClient) ComputeUser24hStats(ctx context.Context, trader string) (KeeperUser24hStats, error) {
	var out KeeperUser24hStats

	const q = `
WITH recent AS (
  SELECT h.*
  FROM perp_trade_history h
  JOIN block b ON b.block = h.block
  WHERE h.trader = $1
    AND b.block_ts >= now() - interval '24 hours'
    AND h.trade_change_type IN ('position_closed', 'position_closed_liquidation')
)
SELECT
  COALESCE(SUM(notional_usd), 0)     AS volume_usd,
  COALESCE(SUM(realized_pnl_usd), 0) AS realized_pnl_usd
FROM recent;
`

	row := kc.db.QueryRowContext(ctx, q, trader)
	if err := row.Scan(&out.VolumeUSD, &out.RealizedPnL); err != nil {
		return KeeperUser24hStats{}, fmt.Errorf("scan keeper stats: %w", err)
	}

	return out, nil
}

func (kc *KeeperClient) Close() error {
	if kc.db != nil {
		return kc.db.Close()
	}
	return nil
}

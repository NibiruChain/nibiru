package evmtrader

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type KeeperClient struct {
	db *sql.DB
}

type KeeperUserAllTimeStats struct {
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

func (kc *KeeperClient) ComputeUserAllTimeStats(ctx context.Context, trader string) (KeeperUserAllTimeStats, error) {
	var out KeeperUserAllTimeStats

	const q = `
SELECT
  volume_usd_cumulative,
  realized_pnl_usd_cumulative
FROM stats_user_portfolio
WHERE user_address = $1
  AND granularity = '1h'
ORDER BY ts DESC
LIMIT 1;
`

	row := kc.db.QueryRowContext(ctx, q, trader)
	if err := row.Scan(&out.VolumeUSD, &out.RealizedPnL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return KeeperUserAllTimeStats{}, nil
		}
		return KeeperUserAllTimeStats{}, fmt.Errorf("scan keeper stats: %w", err)
	}

	return out, nil
}

func (kc *KeeperClient) Close() error {
	if kc.db != nil {
		return kc.db.Close()
	}
	return nil
}

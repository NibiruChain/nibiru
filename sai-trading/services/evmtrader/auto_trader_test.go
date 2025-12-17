package evmtrader

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPositionTracker tests the PositionTracker struct
func TestPositionTracker(t *testing.T) {
	tracker := &PositionTracker{
		TradeIndex:  123,
		OpenBlock:   1000,
		MarketIndex: 0,
	}

	require.Equal(t, uint64(123), tracker.TradeIndex)
	require.Equal(t, uint64(1000), tracker.OpenBlock)
	require.Equal(t, uint64(0), tracker.MarketIndex)
}

// TestAutoTradingConfig tests the AutoTradingConfig struct
func TestAutoTradingConfig(t *testing.T) {
	cfg := AutoTradingConfig{
		MarketIndex:       0,
		CollateralIndex:   1,
		MinTradeSize:      1000000,
		MaxTradeSize:      5000000,
		MinLeverage:       1,
		MaxLeverage:       10,
		BlocksBeforeClose: 10,
		MaxOpenPositions:  5,
		LoopDelaySeconds:  30,
	}

	require.Equal(t, uint64(0), cfg.MarketIndex)
	require.Equal(t, uint64(1), cfg.CollateralIndex)
	require.Equal(t, uint64(1000000), cfg.MinTradeSize)
	require.Equal(t, uint64(5000000), cfg.MaxTradeSize)
	require.Equal(t, uint64(1), cfg.MinLeverage)
	require.Equal(t, uint64(10), cfg.MaxLeverage)
	require.Equal(t, uint64(10), cfg.BlocksBeforeClose)
	require.Equal(t, 5, cfg.MaxOpenPositions)
	require.Equal(t, 30, cfg.LoopDelaySeconds)
}

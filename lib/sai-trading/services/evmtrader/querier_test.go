package evmtrader

import (
	"context"
	"testing"
)

func TestTradeListingCrossesEmptyIndexWindows(t *testing.T) {
	var starts []uint64
	trades, err := collectTradePages(context.Background(), 250, func(_ context.Context, start, limit uint64) ([]Trade, error) {
		starts = append(starts, start)
		if limit != 100 {
			t.Fatalf("scan limit %d", limit)
		}
		switch start {
		case 249:
			return []Trade{{UserTradeIndex: "UserTradeIndex(245)", IsOpen: false}}, nil
		case 149:
			return nil, nil // This entire range was cleaned up.
		case 49:
			return []Trade{{UserTradeIndex: "UserTradeIndex(10)", IsOpen: true, TradeType: "limit"}}, nil
		default:
			t.Fatalf("unexpected cursor %d", start)
			return nil, nil
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(starts) != 3 || len(trades) != 2 || !trades[1].IsOpen {
		t.Fatalf("lost older position: cursors=%v trades=%v", starts, trades)
	}
	_, err = collectTradePages(context.Background(), 0, func(context.Context, uint64, uint64) ([]Trade, error) {
		t.Fatal("queried empty history")
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

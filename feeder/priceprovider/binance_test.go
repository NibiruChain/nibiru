package priceprovider

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBinance(t *testing.T) {
	b, err := DialBinance()
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	p := b.GetPrice("BTCUSDT")
	require.True(t, p.Valid)
	require.Greater(t, p.Price, float64(0))
	require.NotEmpty(t, p.Symbol)
	b.Close()
}

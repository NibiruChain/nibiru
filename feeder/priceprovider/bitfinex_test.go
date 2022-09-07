package priceprovider

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBitfinex(t *testing.T) {
	bfx, err := DialBitfinex([]string{"tBTCUSD", "tETHUSD"})
	require.NoError(t, err)

	time.Sleep(5 * time.Second)
	p1, p2 := bfx.GetPrice("tBTCUSD"), bfx.GetPrice("tETHUSD")

	require.True(t, p1.Valid)
	require.True(t, p2.Valid)
	require.Greater(t, p2.Price, float64(0))
	require.Greater(t, p1.Price, float64(0))
	require.NotEmpty(t, p1.Symbol)
	require.NotEmpty(t, p2.Symbol)

	bfx.Close()
}

package priceprovider

import (
	"github.com/stretchr/testify/require"
	"log"
	"testing"
	"time"
)

func TestNewBinance(t *testing.T) {
	b := NewBinance()
	tick := time.After(30 * time.Second)
	time.Sleep(10 * time.Second)
	for {
		select {
		case <-tick:
			break
		default:
		}
		prices, err := b.GetPrices([]string{"BTCUSDT", "ETHBTC"})
		require.NoError(t, err)
		log.Printf("%#v", prices)
	}
}

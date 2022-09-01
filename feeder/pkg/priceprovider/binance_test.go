package priceprovider

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewBinance(t *testing.T) {
	b, err := DialBinance()
	require.NoError(t, err)
	tick := time.After(30 * time.Second)
	time.Sleep(10 * time.Second)
	for {
		select {
		case <-tick:
			break
		default:
		}
		log.Printf("%#v, %#v", b.GetPrice("BTCUSDT"), b.GetPrice("NOTEXIST"))
	}
}

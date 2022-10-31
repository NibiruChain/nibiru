package priceprovider

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
	"time"
)

var _ io.Writer = (*mockWriter)(nil)

type mockWriter struct {
	w func(p []byte) (n int, err error)
}

func (m mockWriter) Write(p []byte) (n int, err error) { return m.w(p) }

func TestTickSource(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expectedSymbols := []string{"tBTCUSDT"}
		expectedPrices := map[string]float64{"tBTCUSDT": 250_000.56}

		ts := NewTickSource(expectedSymbols, func(symbols []string) (map[string]float64, error) {
			require.Equal(t, expectedSymbols, symbols)
			return expectedPrices, nil
		}, zerolog.New(io.Discard))

		defer ts.Close()

		var gotPrices map[string]SourcePriceUpdate
		select {
		case gotPrices = <-ts.PricesUpdate():
		case <-time.After(6 * time.Second): // timeout
			t.Fatal("timeout when receiving prices")
		}

		require.Equal(t, len(expectedPrices), len(gotPrices))
		for symbol, price := range expectedPrices {
			require.Equal(t, price, gotPrices[symbol].Price)
			require.True(t, time.Now().Sub(gotPrices[symbol].UpdateTime) < 50*time.Millisecond)
		}
	})

	t.Run("price update dropped due to shutdown", func(t *testing.T) {
		// basically every log written ends up here
		logs := new(bytes.Buffer)
		mw := mockWriter{w: func(p []byte) (n int, err error) {
			written, err := logs.Write(p)
			require.NoError(t, err)
			return written, nil
		}}

		expectedSymbols := []string{"tBTCUSDT"}
		expectedPrices := map[string]float64{"tBTCUSDT": 250_000.56}

		ts := NewTickSource(expectedSymbols, func(symbols []string) (map[string]float64, error) {
			return expectedPrices, nil
		}, zerolog.New(mw))

		<-time.After(UpdateTick + 1*time.Second) // wait for a tick update
		ts.Close()                               // make the update be dropped because of close

		require.Contains(t, logs.String(), "dropped price update due to shutdown") // assert logs contained the warning about dropped price updates
	})

	t.Run("logs on price update errors", func(t *testing.T) {
		// basically every log written ends up here
		logs := new(bytes.Buffer)
		mw := mockWriter{w: func(p []byte) (n int, err error) {
			written, err := logs.Write(p)
			require.NoError(t, err)
			return written, nil
		}}

		ts := NewTickSource([]string{"tBTCUSDT"}, func(symbols []string) (map[string]float64, error) {
			return nil, fmt.Errorf("sentinel error")
		}, zerolog.New(mw))
		defer ts.Close()

		<-time.After(UpdateTick + 1*time.Second) // wait for a tick update

		select {
		case <-ts.PricesUpdate():
			t.Fatal("no price updates expected")
		default:
		}

		require.Contains(t, logs.String(), "sentinel error") // assert an error was reported
	})
}

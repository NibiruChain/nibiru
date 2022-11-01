package priceprovider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BitfinexPriceUpdate returns the prices given the symbols or an error.
func BitfinexPriceUpdate(symbols []string) (prices map[string]float64, err error) {
	type ticker []interface{}
	const size = 11
	const lastPriceIndex = 7
	const symbolNameIndex = 0

	resp, err := http.Get("https://api-pub.bitfinex.com/v2/tickers?symbols=" + strings.Join(symbols, ","))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tickers []ticker

	err = json.Unmarshal(b, &tickers)
	if err != nil {
		return nil, err
	}

	prices = make(map[string]float64)
	for _, ticker := range tickers {
		if len(ticker) != size {
			return nil, fmt.Errorf("impossible to parse ticker size %d, %#v", len(ticker), ticker) // TODO(mercilex): return or log and continue?
		}
		tickerName := ticker[symbolNameIndex].(string)
		lastPrice := ticker[lastPriceIndex].(float64)

		prices[tickerName] = lastPrice
	}

	return prices, nil
}

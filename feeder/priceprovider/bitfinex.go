package priceprovider

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	_ PriceProvider = (*Bitfinex)(nil)
)

func DialBitfinex(symbols []string) (PriceProvider, error) {
	bfx := &Bitfinex{
		symbolsQuery: strings.Join(symbols, ","),
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
		rw:           sync.RWMutex{},
		prices:       map[string]priceUpdate{},
		pollTime:     1500 * time.Millisecond,
	}

	go bfx.loop()

	return bfx, nil
}

type Bitfinex struct {
	symbolsQuery string
	stop         chan struct{}
	done         chan struct{}

	pollTime time.Duration

	rw     sync.RWMutex
	prices map[string]priceUpdate
}

func (c *Bitfinex) Close() {
	close(c.stop)
	<-c.done
}

func (c *Bitfinex) loop() {
	defer close(c.done)
	for {
		select {
		case <-c.stop:
			return
		default:
		}

		err := c.updateSymbols()
		if err != nil {
			log.Err(err).Msg("bitfinex failed to update tickers")
		}

		time.Sleep(c.pollTime)
	}
}

func (c *Bitfinex) updateSymbols() (err error) {
	type ticker []interface{}
	const size = 11
	const lastPriceIndex = 7
	const symbolNameIndex = 0

	resp, err := http.Get("https://api-pub.bitfinex.com/v2/tickers?symbols=" + c.symbolsQuery)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var tickers []ticker

	err = json.Unmarshal(b, &tickers)
	if err != nil {
		return err
	}
	c.rw.Lock()
	defer c.rw.Unlock()

	for _, ticker := range tickers {
		if len(ticker) != size {
			return fmt.Errorf("impossible to parse ticker size %d, %#v", len(ticker), ticker) // TODO(mercilex): return or log and continue?
		}

		tickerName := ticker[symbolNameIndex].(string)
		lastPrice := ticker[lastPriceIndex].(float64)

		c.prices[tickerName] = priceUpdate{price: lastPrice, time: time.Now()}
	}

	return nil
}

func (c *Bitfinex) GetPrice(symbol string) PriceResponse {
	c.rw.RLock()
	defer c.rw.RUnlock()

	price, ok := c.prices[symbol]
	return PriceResponse{
		Symbol:         symbol,
		Price:          price.price,
		Valid:          ok,
		Source:         "bitfinex",
		LastUpdateTime: price.time,
	}
}

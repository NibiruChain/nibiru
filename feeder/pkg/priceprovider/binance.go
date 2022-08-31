package priceprovider

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/adshao/go-binance/v2"
)

type Binance struct {
	done chan struct{}
	stop chan struct{}

	rw     sync.RWMutex
	prices map[string]float64
}

func NewBinance() *Binance {
	b := &Binance{rw: sync.RWMutex{}, prices: map[string]float64{}}
	_ = b.loop()
	return b
}

func (b *Binance) GetPrices(symbols []string) ([]Price, error) {
	b.rw.RLock()
	defer b.rw.RUnlock()

	prices := make([]Price, len(symbols))
	for i, symbol := range symbols {
		price, ok := b.prices[symbol]
		if !ok {
			return nil, fmt.Errorf("no prices for: %s", symbol)
		}

		prices[i] = Price{
			Symbol: symbol,
			Price:  price,
		}
	}

	return prices, nil
}

func (b *Binance) loop() error {
	stop, done, err := binance.WsAllMiniMarketsStatServe(func(event binance.WsAllMiniMarketsStatEvent) {
		b.rw.Lock()
		defer b.rw.Unlock()
		for _, symbolEvent := range event {
			price, err := strconv.ParseFloat(symbolEvent.LastPrice, 64)
			if err != nil {
				panic(err)
			}
			b.prices[symbolEvent.Symbol] = price
		}
	}, func(err error) {
		panic(err)
	})

	if err != nil {
		return err
	}

	b.done = done
	b.stop = stop
	return nil
}

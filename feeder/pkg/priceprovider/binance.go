package priceprovider

import (
	"log"
	"strconv"
	"sync"

	"github.com/adshao/go-binance/v2"
)

var (
	_ PriceProvider = (*Binance)(nil)
)

type Binance struct {
	done chan struct{}
	stop chan struct{}

	rw     sync.RWMutex
	prices map[string]float64 // TODO(mercilex): make it a struct which contains the last update time --- over a certain time (ex: 30s) of no updates price is expired
}

func DialBinance() (*Binance, error) {
	b := &Binance{rw: sync.RWMutex{}, prices: map[string]float64{}}
	return b, b.connect()
}

func (b *Binance) GetPrice(symbol string) PriceResponse {
	b.rw.RLock()
	defer b.rw.RUnlock()

	price, ok := b.prices[symbol]
	return PriceResponse{
		Symbol: symbol,
		Price:  price,
		Valid:  ok,
	}
}

func (b *Binance) connect() error {
	stop, done, err := binance.WsAllMiniMarketsStatServe(b.onUpdate, b.onError)

	if err != nil {
		return err
	}

	b.done = done
	b.stop = stop
	return nil
}

func (b *Binance) onUpdate(events binance.WsAllMiniMarketsStatEvent) {
	// process prices strconv conversion
	// before insertion so we can avoid
	// to lock during float parsing time.
	prices := make([]float64, len(events))
	for i, e := range events {
		price, err := strconv.ParseFloat(e.LastPrice, 64)
		if err != nil {
			panic(err)
		}
		prices[i] = price
	}

	// insert blocking
	b.rw.Lock()
	for i, e := range events {
		b.prices[e.Symbol] = prices[i]
	}
	b.rw.Unlock()
}

func (b *Binance) onError(err error) {
	// it is safe to lock here simply because
	// this is called when the writing
	// go-routine has exited.
	// plus the next writer from binance
	// will be running in a different go-routine from this one.
	b.rw.Lock()
	defer b.rw.Unlock()

	log.Printf("binance connection interrupted: %s", err)
	for {
		err := b.connect()
		if err != nil {
			log.Printf("binance reconnection failed: %s", err)
			continue // TODO(mercilex): backoff strategy?
		}
		break
	}
	log.Printf("reconnected to binance")
}

func (b *Binance) Close() {
	panic("implement me") // TODO(mercilex)
}

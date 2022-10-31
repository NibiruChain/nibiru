package priceprovider

import (
	"github.com/rs/zerolog"
	"time"
)

var (
	// PriceTimeout defines after how much time a price is considered expired.
	PriceTimeout = 15 * time.Second
	// UpdateTick defines the wait time between price updates.
	UpdateTick = 3 * time.Second
)

const (
	Bitfinex = "bitfinex"
)

// Source defines a source for price provision.
// This source has no knowledge of nibiru internals
// and mappings across common.AssetPair and the Source
// symbols.
type Source interface {
	// PricesUpdate is a readonly channel which provides
	// the latest prices update. Updates can be provided
	// for one asset only or in batches, hence the map.
	PricesUpdate() <-chan map[string]SourcePriceUpdate
	// Close closes the Source.
	Close()
}

// SourcePriceUpdate defines an update for a symbol for Source implementers.
type SourcePriceUpdate struct {
	Price      float64
	UpdateTime time.Time
}

// TickSourcePriceUpdateFunc is the function used by TickSource to update prices.
// The symbols passed are the symbols we require prices for.
// The returned map must map symbol to its float64 price, or an error.
// If there's a failure in updating only one price then the map can be returned
// without the provided symbol.
type TickSourcePriceUpdateFunc func(symbols []string) (map[string]float64, error)

// NewTickSource instantiates a new TickSource instance, given the symbols and a price updater function
// which returns the latest prices for the provided symbols.
func NewTickSource(symbols []string, updatePricesFunc TickSourcePriceUpdateFunc, log zerolog.Logger) *TickSource {
	ts := &TickSource{
		log:             log,
		stop:            make(chan struct{}),
		done:            make(chan struct{}),
		tick:            time.NewTicker(UpdateTick),
		symbols:         symbols,
		updatePrices:    updatePricesFunc,
		sendPriceUpdate: make(chan map[string]SourcePriceUpdate),
	}
	go ts.loop()
	return ts
}

// TickSource is a Source which updates prices
// every x time.Duration.
type TickSource struct {
	log        zerolog.Logger
	stop, done chan struct{}
	tick       *time.Ticker

	symbols []string

	updatePrices func(symbols []string) (map[string]float64, error)

	sendPriceUpdate chan map[string]SourcePriceUpdate
}

func (s *TickSource) loop() {
	defer s.tick.Stop()
	defer close(s.done)
	for {
		select {
		case <-s.stop:
			return
		case <-s.tick.C:
			now := time.Now()
			s.log.Debug().Time("time", now).Msg("received tick, updating prices")
			prices, err := s.updatePrices(s.symbols)
			if err != nil {
				s.log.Err(err).Msg("failed to update prices")
				break // breaks the current select case, not the for cycle
			}
			update := make(map[string]SourcePriceUpdate, len(prices))
			for symbol, price := range prices {
				update[symbol] = SourcePriceUpdate{
					Price:      price,
					UpdateTime: now,
				}
			}
			s.log.Debug().Msg("sending price update")
			select {
			case s.sendPriceUpdate <- update:
				s.log.Debug().Msg("sent price update")
			case <-s.stop:
				s.log.Warn().Msg("dropped price update due to shutdown")
				return
			}
		}
	}
}

func (s *TickSource) PricesUpdate() <-chan map[string]SourcePriceUpdate {
	return s.sendPriceUpdate
}

func (s *TickSource) Close() {
	close(s.stop)
	<-s.done
}

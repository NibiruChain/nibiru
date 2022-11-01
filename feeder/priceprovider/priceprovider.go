package priceprovider

import (
	"github.com/NibiruChain/nibiru/feeder/types"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/rs/zerolog"
	"sync"
	"time"
)

// NewPriceProvider returns a types.PriceProvider given the price source we want to gather prices from,
// the mapping between nibiru common.AssetPair and the source's symbols, and a zerolog.Logger instance.
func NewPriceProvider(priceSourceName string, pairToSymbolMap map[common.AssetPair]string, log zerolog.Logger) types.PriceProvider {
	var source Source
	switch priceSourceName {
	case Bitfinex:
		source = NewTickSource(symbolsFromPairsToSymbolsMap(pairToSymbolMap), BitfinexPriceUpdate, log)
	default:
		panic("unknown price provider: " + priceSourceName)
	}
	return newPriceProvider(source, priceSourceName, pairToSymbolMap, log)
}

// newPriceProvider returns a raw *PriceProvider given a Source implementer, the source name and the
// map of nibiru common.AssetPair to Source's symbols, plus the zerolog.Logger instance.
// Exists for testing purposes.
func newPriceProvider(source Source, sourceName string, pairToSymbolsMap map[common.AssetPair]string, log zerolog.Logger) *PriceProvider {
	log = log.With().
		Str("component", "price-provider").
		Str("price-source", sourceName).
		Logger()

	pp := &PriceProvider{
		log:          log,
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
		source:       source,
		sourceName:   sourceName,
		pairToSymbol: pairToSymbolsMap,
		mu:           sync.Mutex{},
		lastPrices:   map[string]SourcePriceUpdate{},
	}
	go pp.loop()
	return pp
}

// PriceProvider implements the types.PriceProvider interface.
// it wraps a Source and handles conversions between
// nibiru asset pair to exchange symbols.
type PriceProvider struct {
	log zerolog.Logger

	stop, done chan struct{}

	source       Source
	sourceName   string
	pairToSymbol map[common.AssetPair]string

	mu         sync.Mutex
	lastPrices map[string]SourcePriceUpdate
}

// GetPrice returns the types.Price for the given common.AssetPair
// in case price has expired, or for some reason it's impossible to
// get the last available price, then an invalid types.Price is returned.
func (p *PriceProvider) GetPrice(pair common.AssetPair) types.Price {
	symbol, ok := p.pairToSymbol[pair]
	// in case this is an unknown symbol, which might happen
	// when for example we have a param update, then we return
	// an abstain vote on the provided asset pair.
	if !ok {
		p.log.Warn().Str("nibiru-pair", pair.String()).Msg("unknown nibiru pair")
		return types.Price{
			Pair:   pair,
			Price:  0,
			Source: p.sourceName,
			Valid:  false,
		}
	}
	p.mu.Lock()
	price, ok := p.lastPrices[symbol]
	p.mu.Unlock()
	return types.Price{
		Pair:   pair,
		Price:  price.Price,
		Source: p.sourceName,
		Valid:  isValid(price, ok),
	}
}

func (p *PriceProvider) loop() {
	defer close(p.done)
	defer p.source.Close()

	for {
		select {
		case <-p.stop:
			return
		case updates := <-p.source.PricesUpdate():
			p.mu.Lock()
			for symbol, price := range updates {
				p.lastPrices[symbol] = price
			}
			p.mu.Unlock()
		}
	}
}

func (p *PriceProvider) Close() {
	close(p.stop)
	<-p.done
}

// symbolsFromPairsToSymbolsMap returns the symbols list
// given the map which maps nibiru chain pairs to exchange symbols.
func symbolsFromPairsToSymbolsMap(m map[common.AssetPair]string) []string {
	symbols := make([]string, 0, len(m))
	for _, v := range m {
		symbols = append(symbols, v)
	}
	return symbols
}

// isValid is a helper function which asserts if a price is valid given
// if it was found and the time at which it was last updated.
func isValid(price SourcePriceUpdate, found bool) bool {
	return found && time.Now().Sub(price.UpdateTime) < PriceTimeout
}

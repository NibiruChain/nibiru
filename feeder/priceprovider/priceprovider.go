package priceprovider

import (
	"fmt"
	"github.com/rs/zerolog/log"
)

// PriceResponse defines the response given by
// PriceProvider implementers when asked for prices.
// Symbol must always be not-empty.
// TODO(mercilex): maybe add last update time here
type PriceResponse struct {
	// Symbol defines the symbol.
	Symbol string
	// Price defines the price of Symbol.
	Price float64
	// Valid defines if the price can be used or not.
	Valid bool
	// Source defines the source of the price, optional.
	Source string
}

// PriceProvider defines a price provider's behavior.
type PriceProvider interface {
	// GetPrice returns the PriceResponse
	// for the given symbol. PriceResponse.Symbol
	// must always be non-empty.
	// If there are some errors PriceResponse.Valid must be set to false,
	// and must be true in case everything went fine.
	GetPrice(symbol string) PriceResponse
	Close()
}

var _ PriceProvider = (*ExchangeToChainSymbolPriceProvider)(nil)

// NewExchangeToChainSymbolPriceProvider returns a new ExchangeToChainSymbolPriceProvider instance
// given a price provider and the chain to exchange symbols map.
func NewExchangeToChainSymbolPriceProvider(pp PriceProvider, chainToExchangeSymbolsMap map[string]string) PriceProvider {
	return ExchangeToChainSymbolPriceProvider{
		kind:            fmt.Sprintf("%T", pp),
		pp:              pp,
		chainToExchange: chainToExchangeSymbolsMap,
	}
}

// ExchangeToChainSymbolPriceProvider implement PriceProvider and
// wraps a PriceProvider implementer, when asking for prices
// chain symbols are converted to exchange symbols, and when
// return PriceResponse the exchange symbol is converted back
// to chain symbol.
type ExchangeToChainSymbolPriceProvider struct {
	pp              PriceProvider     // the original price provider
	chainToExchange map[string]string // maps chain to exchange symbols
	kind            string
}

// GetPrice converts the chain symbol to exchange symbol and queries
// the wrapped PriceProvider for the price.
// If the symbol is unknown then an invalid PriceResponse is returned,
// otherwise a valid PriceResponse is returned with its symbol being
// the chain symbol.
func (e ExchangeToChainSymbolPriceProvider) GetPrice(chainSymbol string) PriceResponse {
	exchangeSymbol, ok := e.chainToExchange[chainSymbol]
	if !ok {
		log.
			Warn().
			Str("price provider", e.kind).
			Str("chain symbol", chainSymbol).
			Msg("chain to exchange symbol not found")
		return PriceResponse{
			Symbol: chainSymbol,
			Price:  0,
			Valid:  false, // signal price is not ok
		}
	}

	price := e.pp.GetPrice(exchangeSymbol)
	price.Symbol = chainSymbol
	return price
}

func (e ExchangeToChainSymbolPriceProvider) Close() {
	e.pp.Close()
}

var _ PriceProvider = (*AggregatePriceProvider)(nil)

// NewAggregatePriceProvider instantiates a new AggregatePriceProvider instance
// given multiple PriceProvider.
func NewAggregatePriceProvider(pps []PriceProvider) PriceProvider {
	a := AggregatePriceProvider{make(map[int]PriceProvider, len(pps))}
	for i, pp := range pps {
		a.pps[i] = pp
	}
	return a
}

// AggregatePriceProvider aggregates multiple price providers
// and queries them for prices.
type AggregatePriceProvider struct {
	pps map[int]PriceProvider // we use a map here to provide random ranging (since golang's map range is unordered)
}

// GetPrice fetches the first available and correct price from the wrapped PriceProviders.
// Iteration is exhaustive and random.
// If no correct PriceResponse is found, then an invalid PriceResponse is returned.
func (a AggregatePriceProvider) GetPrice(symbol string) PriceResponse {
	// iterate randomly, if we find a valid price, we return it
	// otherwise we go onto the next PriceProvider to ask for prices.
	for _, pp := range a.pps {
		price := pp.GetPrice(symbol)
		if price.Valid {
			return price
		}
	}

	// if we reach here no valid symbols were found
	return PriceResponse{
		Symbol: symbol,
		Price:  0,
		Valid:  false,
	}
}

func (a AggregatePriceProvider) Close() {
	for _, pp := range a.pps {
		pp.Close()
	}
}

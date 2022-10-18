package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

// Params is the x/oracle specific subset of parameters required for price feeding.
type Params struct {
	// Symbols are the symbols we need to provide prices for.
	Symbols []common.AssetPair
	// VotePeriodBlocks is how
	VotePeriodBlocks uint64
}

// VotingPeriod contains information
// concerning the current voting period.
type VotingPeriod struct {
	// Height is the height of the voting period.
	Height uint64
}

// ValidatorSet is a helper type containing the active validator set.
type ValidatorSet map[string]struct{}

func (v ValidatorSet) Has(val types.ValAddress) bool {
	_, has := v[val.String()]
	return has
}

func (v ValidatorSet) Remove(val types.ValAddress) {
	s := val.String()
	if _, exists := v[s]; !exists {
		panic("trying to Remove validator which is not part of the current set: " + s)
	}
	delete(v, val.String())
}

func (v ValidatorSet) Insert(val types.ValAddress) {
	s := val.String()
	if _, exists := v[s]; exists {
		panic("trying to Insert validator which is already part of the current set: " + s)
	}
	v[val.String()] = struct{}{}
}

// ValidatorSetChanges contains
// the validator set updates.
type ValidatorSetChanges struct {
	// In contains validators which joined the active set.
	In []types.ValAddress
	// Out contains validators which exited the active set.
	Out []types.ValAddress
}

// EventsStream defines the asynchronous stream
// of events required by the feeder's Loop function.
//
//go:generate mockgen --destination ./mocks/feeder/types/events_stream.go . EventsStream
type EventsStream interface {
	// ParamsUpdate signals a new Params update.
	// EventsStream must provide, on startup, the
	// initial Params found on the chain.
	ParamsUpdate() <-chan Params
	// VotingPeriodStarted signals a new x/oracle
	// voting period has just started.
	VotingPeriodStarted() <-chan VotingPeriod
	// ValidatorSetChanged signals a change in the validator set.
	// EventsStream must provide, on startup,
	// the initial ValidatorSetChanges.
	ValidatorSetChanged() <-chan ValidatorSetChanges
	// Close shuts down the EventsStream.
	Close()
}

// Price defines the price of a symbol.
type Price struct {
	// Symbol defines the symbol we're posting prices for.
	Symbol common.AssetPair
	// Price defines the symbol's price.
	Price float64
	// Source defines the source which is providing the prices.
	Source string
	// Valid reports whether the price is valid or not.
	// If not valid then an abstain vote will be posted.
	Valid bool
}

// PriceProvider defines an exchange API
// which provides prices for the given assets.
//
//go:generate mockgen --destination ./mocks/feeder/types/price_provider.go . PriceProvider
type PriceProvider interface {
	// GetPrice returns the Price for the given symbol.
	// Price.Symbol, Price.Source must always be non-empty.
	// If there are errors whilst fetching prices, then
	// Price.Valid must be set to false.
	GetPrice(symbol common.AssetPair) Price
	// Close shuts down the PriceProvider.
	Close()
}

// PricePoster defines the validator oracle client,
// which sends new prices.
//
//go:generate mockgen --destination ./mocks/feeder/types/price_poster.go . PricePoster
type PricePoster interface {
	// Whoami returns the validator address the PricePoster
	// is sending prices for.
	Whoami() types.ValAddress
	// SendPrices sends the provided slice of Price.
	// It must keep trying to send the prices until it
	// either succeeds or the provided context.Context
	// is canceled. The operation must not be blocking.
	// It returns a done channel which must be closed after prices
	// are successfully sent to the chain.
	SendPrices(ctx context.Context, prices []Price) (done chan struct{})
	// Close shuts down the PricePoster.
	Close()
}

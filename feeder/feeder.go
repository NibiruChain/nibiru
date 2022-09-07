package feeder

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/NibiruChain/nibiru/feeder/priceprovider"
)

// Params is the x/oracle specific subset of parameters required for price feeding.
type Params struct {
	// Symbols are the symbols we need to provide prices for.
	Symbols []string
	// VotePeriodBlocks is how
	VotePeriodBlocks uint64
}

// VotingPeriod contains information
// concerning the current voting period.
type VotingPeriod struct {
	// Height is the height of the voting period.
	Height uint64
}

// ValidatorSetChanges contains
// the validator set updates.
type ValidatorSetChanges struct {
	// In contains validators which joined the active set.
	In []sdk.ValAddress
	// Out contains validators which exited the active set.
	Out []sdk.ValAddress
}

// EventsStream defines an interface which emits a stream
// of events from the chain with the x/oracle module.
type EventsStream interface {
	// ParamsUpdate signals a new Params update.
	ParamsUpdate() <-chan Params
	// NewVotingPeriod signals when a new voting period starts.
	NewVotingPeriod() <-chan VotingPeriod
	// ValidatorSetChanges signals when changes happen in
	// the validator set.
	ValidatorSetChanges() <-chan ValidatorSetChanges
	// Close shuts down the EventsStream.
	Close()
}

// SymbolPrice defines the symbol of a price
type SymbolPrice struct {
	Symbol string
	Price  float64
	Source string
}

type TxClient interface {
	SendPrices(prices []SymbolPrice)
	Close()
}

func Dial(tx TxClient, es EventsStream, pp priceprovider.PriceProvider) (*Feeder, error) {
	return &Feeder{
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
		params: Params{},
		tx:     tx,
		events: es,
		pp:     pp,
	}, nil
}

type Feeder struct {
	stop chan struct{}
	done chan struct{}

	params Params

	tx     TxClient
	events EventsStream
	pp     priceprovider.PriceProvider
}

func (f *Feeder) Run() {
	defer close(f.done)

	log.Debug().Msg("waiting initial parameters")
	select {
	case params := <-f.events.ParamsUpdate():
		f.params = params
		log.Debug().Interface("initial params", params).Msg("got initial params")
	case <-time.After(15 * time.Second):
		panic("timeout whilst fetching initial params")
	}

	for {
		select {
		case <-f.stop:
			return
		case params := <-f.events.ParamsUpdate():
			log.Info().Interface("params update", params).Msg("received new params update")
			f.params = params
		case votePeriod := <-f.events.NewVotingPeriod():
			log.Info().
				Uint64("voting period", votePeriod.Height/f.params.VotePeriodBlocks).
				Uint64("voting period start block", votePeriod.Height).
				Msg("new voting period started")

			log.Debug().Msg("fetching prices")
			prices := make([]SymbolPrice, len(f.params.Symbols))
			for i, symbol := range f.params.Symbols {
				price := f.pp.GetPrice(symbol)
				if !price.Valid {
					log.Warn().Str("symbol", symbol).Msg("no valid prices for symbol")
				}

				if price.Symbol == "" {
					panic("bad implementation of price provider interface")
				}

				prices[i] = SymbolPrice{
					Symbol: symbol,
					Price:  price.Price,
					Source: price.Source,
				}
			}
			log.Info().Interface("prices", prices).Msg("posting prices")
			f.tx.SendPrices(prices) // TODO(mercilex): add a give up strategy which does not block us for too much time and does not make us miss multiple voting periods
		}
	}
}

func (f *Feeder) Close() {
	close(f.stop)
	<-f.done
	f.tx.Close()
	f.events.Close()
	f.pp.Close()
}

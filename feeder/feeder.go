package feeder

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/NibiruChain/nibiru/feeder/oracle"
	"github.com/NibiruChain/nibiru/feeder/priceprovider"
)

func Dial(c Config) (*Feeder, error) {
	tx, err := oracle.NewTxClient(c.GRPCEndpoint, c.Validator, c.Feeder, c.Cache, c.KeyRing, c.ChainID)
	if err != nil {
		return nil, err
	}

	events, err := oracle.NewEventsClient(c.TendermintWebsocketEndpoint, c.GRPCEndpoint)
	if err != nil {
		return nil, err
	}

	pp, err := PriceProviderFromChainToExchangeSymbols(c.ChainToExchangeSymbols)
	if err != nil {
		return nil, err
	}

	return &Feeder{
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
		params: oracle.ParamsUpdate{},
		tx:     tx,
		events: events,
		pp:     pp,
	}, nil
}

type Feeder struct {
	stop chan struct{}
	done chan struct{}

	params oracle.ParamsUpdate

	tx     *oracle.TxClient
	events *oracle.EventsClient
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
		case height := <-f.events.NewVotingPeriod():
			log.Info().
				Uint64("voting period", height/f.params.VotePeriodBlocks).
				Uint64("voting period start block", height).
				Msg("new voting period started")

			log.Debug().Msg("fetching prices")
			prices := make([]oracle.SymbolPrice, len(f.params.Symbols))
			for i, symbol := range f.params.Symbols {
				price := f.pp.GetPrice(symbol)
				if !price.Valid {
					log.Warn().Str("symbol", symbol).Msg("no valid prices for symbol")
				}

				if price.Symbol == "" {
					panic("bad implementation of price provider interface")
				}

				prices[i] = oracle.SymbolPrice{
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

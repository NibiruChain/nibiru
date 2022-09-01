package feeder

import (
	"github.com/NibiruChain/nibiru/feeder/oracle"
	"github.com/NibiruChain/nibiru/feeder/priceprovider"
	"github.com/rs/zerolog/log"
)

func Dial(c Config) (*Feeder, error) {
	tx, err := oracle.NewTxClient(c.GRPCEndpoint, c.Validator, c.Feeder, c.Cache, c.KeyRing)
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
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
		symbols: nil,
		tx:      tx,
		events:  events,
		pp:      pp,
	}, nil
}

type Feeder struct {
	stop chan struct{}
	done chan struct{}

	symbols []string

	tx     *oracle.TxClient
	events *oracle.EventsClient
	pp     priceprovider.PriceProvider
}

func (f *Feeder) Run() {
	defer close(f.done)
	for {
		select {
		case <-f.stop:
			return
		case newSymbols := <-f.events.SymbolsUpdate():
			log.Info().Strs("symbols", newSymbols).Msg("received new symbols update")
			f.symbols = newSymbols
		case height := <-f.events.NewVotingPeriod():
			log.Info().Uint64("voting period", height).Msg("new voting period started")

			prices := make([]oracle.SymbolPrice, len(f.symbols))
			for i, symbol := range f.symbols {
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
				}
			}
			log.Info().Interface("prices", prices).Msg("posting prices")
			f.tx.SendPrices(prices) // TODO(mercilex): add a give up strategy which does not block us for too much time znd does not make us miss multiple voting periods
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

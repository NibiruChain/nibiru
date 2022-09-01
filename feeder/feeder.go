package feeder

import (
	"github.com/NibiruChain/nibiru/feeder/pkg/oracle"
	"github.com/NibiruChain/nibiru/feeder/pkg/priceprovider"
	"log"
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
	events oracle.EventsClient
	pp     priceprovider.PriceProvider
}

func (f *Feeder) Run() {
	defer close(f.done)
	for {
		select {
		case <-f.stop:
			return
		case newSymbols := <-f.events.SymbolsUpdate():
			log.Printf("received new symbols update: %#v", newSymbols)
			f.symbols = newSymbols
		case height := <-f.events.NewVotingPeriod():
			log.Printf("new voting period for height: %d", height)

			prices := make([]oracle.SymbolPrice, len(f.symbols))
			for i, symbol := range f.symbols {
				price := f.pp.GetPrice(symbol)
				if !price.Valid {
					log.Printf("no valid prices for: %s", symbol)
				}

				if price.Symbol == "" {
					panic("bad implementation of price provider interface")
				}

				prices[i] = oracle.SymbolPrice{
					Symbol: symbol,
					Price:  price.Price,
				}
			}
			log.Printf("posting prices: %#v", prices)
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

package feeder

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/NibiruChain/nibiru/feeder/types"
)

var (
	InitTimeout = 5 * time.Second
)

// votingPeriodContext keeps track of the current voting period.
type votingPeriodContext struct {
	height                     uint64
	votingPeriodDurationBlocks uint64
	cancelSendPrices           func() // signals cancel send prices for types.PricePoster
	pricePosterCtx             context.Context
	sendPricesDone             chan struct{} // response given by types.PricePoster to signal if prices were sent correctly.
}

// Feeder is the price feeder.
type Feeder struct {
	log zerolog.Logger

	stop chan struct{}
	done chan struct{}

	params types.Params

	votingPeriodContext *votingPeriodContext

	eventsStream  types.EventsStream
	pricePoster   types.PricePoster
	priceProvider types.PriceProvider
}

// Run instantiates a new Feeder instance.
func Run(stream types.EventsStream, poster types.PricePoster, provider types.PriceProvider, log zerolog.Logger) *Feeder {
	f := &Feeder{
		log:                 log,
		stop:                make(chan struct{}),
		done:                make(chan struct{}),
		params:              types.Params{},
		votingPeriodContext: nil,
		eventsStream:        stream,
		pricePoster:         poster,
		priceProvider:       provider,
	}

	// init params
	select {
	case initParams := <-stream.ParamsUpdate():
		f.handleParamsUpdate(initParams)
	case <-time.After(InitTimeout):
		panic("init timeout deadline exceeded")
	}

	go f.loop()

	return f
}

func (f *Feeder) loop() {
	defer close(f.done)
	defer f.eventsStream.Close()
	defer f.pricePoster.Close()
	defer f.priceProvider.Close()
	defer f.endLastVotingPeriod()
	for {
		select {
		case <-f.stop:
			return
		case params := <-f.eventsStream.ParamsUpdate():
			f.log.Info().Interface("changes", params).Msg("params changed")
			f.handleParamsUpdate(params)
		case vp := <-f.eventsStream.VotingPeriodStarted():
			f.log.Info().Interface("voting-period", vp).Msg("new voting period")
			f.handleVotingPeriod(vp)
		}
	}
}

func (f *Feeder) handleParamsUpdate(params types.Params) {
	f.params = params
}

func (f *Feeder) handleVotingPeriod(vp types.VotingPeriod) {
	f.endLastVotingPeriod()
	f.startNewVotingPeriod(vp)
}

func (f *Feeder) endLastVotingPeriod() {
	if f.votingPeriodContext == nil {
		return
	}

	// cancel the last voting period send prices operation
	f.votingPeriodContext.cancelSendPrices()
	// check if it prices sends went fine
	select {
	case <-f.votingPeriodContext.sendPricesDone:
	default:
		// TODO(testinginprod): from coverage we can see this was called. Maybe in the future
		// we can add something extra to assert this was hit in testing.
		f.log.Err(fmt.Errorf("vote period missed")).
			Uint64("voting-period-start-height", f.votingPeriodContext.height).
			Uint64("voting-period-block-duration", f.votingPeriodContext.votingPeriodDurationBlocks)
	}
	// reset voting period context
	f.votingPeriodContext = nil
}

func (f *Feeder) startNewVotingPeriod(vp types.VotingPeriod) {
	// gather prices
	prices := make([]types.Price, len(f.params.Pairs))
	for i, p := range f.params.Pairs {
		price := f.priceProvider.GetPrice(p)
		if !price.Valid {
			f.log.Err(fmt.Errorf("no valid price")).Str("asset", p.String()).Str("source", price.Source)
			price.Price = 0
		}
		prices[i] = price
	}

	// send prices asynchronously and non-blocking
	ctx, cancel := context.WithCancel(context.Background())
	asyncSendPriceResponse := f.pricePoster.SendPrices(ctx, prices)

	f.votingPeriodContext = &votingPeriodContext{
		height:                     vp.Height,
		votingPeriodDurationBlocks: f.params.VotePeriodBlocks,
		cancelSendPrices:           cancel,
		pricePosterCtx:             ctx,
		sendPricesDone:             asyncSendPriceResponse,
	}
}

func (f *Feeder) Close() {
	close(f.stop)
	<-f.done
}

package oracle

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/NibiruChain/nibiru/feeder/websocket"

	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

var (
	Timeout = websocket.Timeout
)

type ParamsUpdate struct {
	// Symbols indicates the symbols oracles need to provide prices for.
	Symbols []string
	// VotePeriodBlocks indicates the number of blocks between each voting period.
	VotePeriodBlocks uint64
}

func NewEventsClient(tendermintEndpoint string, grpcEndpoint string) (*EventsClient, error) {
	ec := &EventsClient{
		tm:              tendermintEndpoint,
		grpc:            grpcEndpoint,
		paramsUpdate:    make(chan ParamsUpdate, 1), // it has one as buffer for the initial params
		newVotingPeriod: make(chan uint64),
	}

	return ec, ec.init()
}

// EventsClient is a client that keeps track, asynchronously,
// of chain updates using the tendermint websocket.
type EventsClient struct {
	tm   string
	grpc string

	votingPeriod uint64

	paramsUpdate    chan ParamsUpdate
	newVotingPeriod chan uint64

	done chan struct{}
	stop chan struct{}
}

// init initializes the client by getting
// the initial oracle parameters and connecting
// to the tendermint websocket.
func (c *EventsClient) init() error {
	err := c.updateParams()
	if err != nil {
		return err
	}
	err = c.connectWebsocket()
	if err != nil {
		return err
	}

	return nil
}

// connectWebsocket connects to the tendermint websocket.
func (c *EventsClient) connectWebsocket() error {
	const message = `{"jsonrpc":"2.0","method":"subscribe","id":0,"params":{"query":"tm.event='NewBlock'"}}`
	done, stop, err := websocket.NewJSON(c.tm, json.RawMessage(message), c.onNewBlock, c.onWsError)
	if err != nil {
		return err
	}

	c.done = done
	c.stop = stop

	return nil
}

// onNewBlock handles the logic of handling new block events.
func (c *EventsClient) onNewBlock(msg newBlockJSON) {
	// init msg
	if msg.Result.Data.Value.Block.Header.Height == "" {
		return
	}

	c.signalNewVoting(msg.Result.Data.Value.Block.Header.Height)
}

// onWsError is the error handler and it attempts to reconnect to the tendermint websocket.
func (c *EventsClient) onWsError(err error) {
	log.Error().Err(err).Msg("events client websocket error")
	log.Info().Msg("attempting events client reconnection")
	for {
		select {
		case <-c.done:
			return
		default:
		}
		err := c.connectWebsocket()
		if err == nil {
			break
		}
		log.Error().Err(err).Msg("events client reconnection error")
		time.Sleep(5 * time.Second) // TODO(mercilex): backoff
	}
	log.Info().Msg("events client reconnected")
}

func (c *EventsClient) NewVotingPeriod() <-chan uint64 {
	return c.newVotingPeriod
}

func (c *EventsClient) ParamsUpdate() (newSymbols <-chan ParamsUpdate) {
	return c.paramsUpdate
}

func (c *EventsClient) Close() {
	// TODO(mercilex): this might cause a race condition in case we're reconnecting and close is called as reconnection is happening
	close(c.stop)
	<-c.done
}

// signalNewVoting signals a new voting period in case
// the provided heightStr matches the last block of the
// current voting period.
func (c *EventsClient) signalNewVoting(heightStr string) {
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		panic(err)
	}

	// this basically checks if the current block + 1 is the next voting period
	// we check current block + 1 because it means that from next block onwards
	// we can unveil our votes and insert the new ones
	if (height+1)%c.votingPeriod == 0 {
		c.newVotingPeriod <- height + 1 // signal
	}
}

func (c *EventsClient) updateParams() error {
	conn, err := grpc.Dial(c.grpc, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	oracle := types.NewQueryClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	p, err := oracle.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		return err
	}

	c.votingPeriod = p.Params.VotePeriod

	targets, err := oracle.VoteTargets(ctx, &types.QueryVoteTargetsRequest{})
	if err != nil {
		return err
	}

	c.paramsUpdate <- ParamsUpdate{
		Symbols:          targets.VoteTargets,
		VotePeriodBlocks: p.Params.VotePeriod,
	}
	return nil
}

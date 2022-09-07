package oracle

import (
	"context"
	"encoding/json"
	"github.com/NibiruChain/nibiru/feeder"
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

var (
	_ feeder.EventsStream = (*Stream)(nil)
)

// NewStream instantiates a new *Stream instance.
func NewStream(tendermintEndpoint string, grpcEndpoint string) (*Stream, error) {
	ec := &Stream{
		tm:                  tendermintEndpoint,
		grpc:                grpcEndpoint,
		paramsUpdate:        make(chan feeder.Params, 1), // it has one as buffer for the initial params
		newVotingPeriod:     make(chan feeder.VotingPeriod),
		validatorSetChanges: make(chan feeder.ValidatorSetChanges, 1), // it has one as buffer for the initial params
	}

	return ec, ec.init()
}

// Stream is a client that keeps track, asynchronously,
// of chain updates using the tendermint websocket.
type Stream struct {
	tm   string
	grpc string

	votingPeriod uint64

	paramsUpdate        chan feeder.Params
	newVotingPeriod     chan feeder.VotingPeriod
	validatorSetChanges chan feeder.ValidatorSetChanges

	done chan struct{}
	stop chan struct{}
}

func (c *Stream) ValidatorSetChanges() <-chan feeder.ValidatorSetChanges {
	return c.validatorSetChanges
}

// init initializes the client by getting
// the initial oracle parameters and connecting
// to the tendermint websocket.
func (c *Stream) init() error {
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
func (c *Stream) connectWebsocket() error {
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
func (c *Stream) onNewBlock(msg newBlockJSON) {
	// init msg
	if msg.Result.Data.Value.Block.Header.Height == "" {
		return
	}

	c.signalNewVoting(msg.Result.Data.Value.Block.Header.Height)
}

// onWsError is the error handler and it attempts to reconnect to the tendermint websocket.
func (c *Stream) onWsError(err error) {
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

func (c *Stream) NewVotingPeriod() <-chan feeder.VotingPeriod {
	return c.newVotingPeriod
}

func (c *Stream) ParamsUpdate() (newSymbols <-chan feeder.Params) {
	return c.paramsUpdate
}

func (c *Stream) Close() {
	// TODO(mercilex): this might cause a race condition in case we're reconnecting and close is called as reconnection is happening
	close(c.stop)
	<-c.done
}

// signalNewVoting signals a new voting period in case
// the provided heightStr matches the last block of the
// current voting period.
func (c *Stream) signalNewVoting(heightStr string) {
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		panic(err)
	}

	// this basically checks if the current block + 1 is the next voting period
	// we check current block + 1 because it means that from next block onwards
	// we can unveil our votes and insert the new ones
	if (height+1)%c.votingPeriod == 0 {
		c.newVotingPeriod <- feeder.VotingPeriod{Height: height + 1}
	}
}

func (c *Stream) updateParams() error {
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

	c.paramsUpdate <- feeder.Params{
		Symbols:          targets.VoteTargets,
		VotePeriodBlocks: p.Params.VotePeriod,
	}
	return nil
}

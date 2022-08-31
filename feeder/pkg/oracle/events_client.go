package oracle

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/feeder/pkg/websocket"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

var (
	Timeout = websocket.Timeout
)

type EventsClient interface {
	SymbolsUpdate() (newSymbols <-chan []string)
	NewVotingPeriod() (height <-chan uint64)
	Close()
}

func NewEventsClient(tendermintEndpoint string, grpcEndpoint string) (EventsClient, error) {
	ec := &eventsClient{
		tm:              tendermintEndpoint,
		grpc:            grpcEndpoint,
		symbolsUpdate:   make(chan []string, 1), // it has one as buffer for the initial vote targets
		newVotingPeriod: make(chan uint64),
	}

	return ec, ec.init()
}

var (
	_ EventsClient = (*eventsClient)(nil)
)

type eventsClient struct {
	tm   string
	grpc string

	votingPeriod uint64

	symbolsUpdate   chan []string
	newVotingPeriod chan uint64
}

func (c *eventsClient) init() error {
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
func (c *eventsClient) connectWebsocket() error {
	const message = `{"jsonrpc":"2.0","method":"subscribe","id":0,"params":{"query":"tm.event='NewBlock'"}}`
	_, _, err := websocket.NewJSON(c.tm, json.RawMessage(message), c.onNewBlock, c.onWsError) // TODO(mercilex): stop strategy
	if err != nil {
		return err
	}

	return nil
}

func (c *eventsClient) onNewBlock(msg newBlockJSON) {
	// init msg
	if msg.Result.Data.Value.Block.Header.Height == "" {
		return
	}

	c.signalNewVoting(msg.Result.Data.Value.Block.Header.Height)
}

func (c *eventsClient) onWsError(err error) {
	log.Printf("error received from websocket: %s", err)
	log.Printf("attempting reconnection")
	for {
		err := c.connectWebsocket()
		if err == nil {
			break
		}
		log.Printf("error whilst attempting to reconnect: %s", err)
		time.Sleep(5 * time.Second) // TODO(mercilex): custom reconnect strategy?
	}
}

func (c *eventsClient) NewVotingPeriod() <-chan uint64 {
	return c.newVotingPeriod
}

func (c *eventsClient) SymbolsUpdate() (newSymbols <-chan []string) {
	return c.symbolsUpdate
}

func (c *eventsClient) Close() {
	//TODO implement me
	panic("implement me")
}

// signalNewVoting signals a new voting period in case
// the provided heightStr matches the last block of the
// current voting period.
func (c *eventsClient) signalNewVoting(heightStr string) {
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

func (c *eventsClient) updateParams() error {
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

	c.symbolsUpdate <- targets.VoteTargets
	return nil
}

package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/abci/types"
	"time"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/testutil"
)

type increaseBlockNumberBy struct {
	numBlocks int64
}

func (i increaseBlockNumberBy) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	app.EndBlocker(ctx, types.RequestEndBlock{Height: ctx.BlockHeight()})

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + i.numBlocks)

	return ctx, nil
}

// IncreaseBlockNumberBy increases the block number by the given number of blocks
func IncreaseBlockNumberBy(numBlocks int64) testutil.Action {
	return increaseBlockNumberBy{numBlocks: numBlocks}
}

type increaseBlockTimeBy struct {
	numSeconds time.Duration
}

func (i increaseBlockTimeBy) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * i.numSeconds))

	return ctx, nil
}

func IncreaseBlockTimeBy(numSeconds time.Duration) testutil.Action {
	return increaseBlockTimeBy{numSeconds: numSeconds}
}

type setBlockTime struct {
	blockTime time.Time
}

func (s setBlockTime) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	return ctx.WithBlockTime(s.blockTime), nil
}

// SetBlockTime sets the block time to the given value
func SetBlockTime(blockTime time.Time) testutil.Action {
	return setBlockTime{blockTime: blockTime}
}

type setBlockNumber struct {
	blockNumber int64
}

func (s setBlockNumber) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	return ctx.WithBlockHeight(s.blockNumber), nil
}

// SetBlockNumber sets the block number to the given value
func SetBlockNumber(blockNumber int64) testutil.Action {
	return setBlockNumber{blockNumber: blockNumber}
}

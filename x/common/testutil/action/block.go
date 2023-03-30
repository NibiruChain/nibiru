package action

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/app"
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
func IncreaseBlockNumberBy(numBlocks int64) Action {
	return increaseBlockNumberBy{numBlocks: numBlocks}
}

type increaseBlockTimeBy struct {
	seconds time.Duration
}

func (i increaseBlockTimeBy) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * i.seconds))

	return ctx, nil
}

func IncreaseBlockTimeBy(seconds time.Duration) Action {
	return increaseBlockTimeBy{seconds: seconds}
}

type setBlockTime struct {
	blockTime time.Time
}

func (s setBlockTime) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	return ctx.WithBlockTime(s.blockTime), nil
}

// SetBlockTime sets the block time to the given value
func SetBlockTime(blockTime time.Time) Action {
	return setBlockTime{blockTime: blockTime}
}

type setBlockNumber struct {
	blockNumber int64
}

func (s setBlockNumber) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	return ctx.WithBlockHeight(s.blockNumber), nil
}

// SetBlockNumber sets the block number to the given value
func SetBlockNumber(blockNumber int64) Action {
	return setBlockNumber{blockNumber: blockNumber}
}

type moveToNextBlock struct{}

func (m moveToNextBlock) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	app.EndBlock(types.RequestEndBlock{})
	app.Commit()

	app.BeginBlock(types.RequestBeginBlock{
		Header: tmproto.Header{Height: ctx.BlockHeight() + 1},
	})

	return app.NewContext(
		false,
		tmproto.Header{Height: ctx.BlockHeight() + 1},
	).WithBlockTime(ctx.BlockTime().Add(time.Second * 5)), nil
}

func MoveToNextBlock() Action {
	return moveToNextBlock{}
}

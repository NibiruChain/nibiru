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

func (i increaseBlockNumberBy) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	app.EndBlocker(ctx, types.RequestEndBlock{Height: ctx.BlockHeight()})

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + i.numBlocks)

	return ctx, nil, true
}

// IncreaseBlockNumberBy increases the block number by the given number of blocks
func IncreaseBlockNumberBy(numBlocks int64) Action {
	return increaseBlockNumberBy{numBlocks: numBlocks}
}

type increaseBlockTimeBy struct {
	seconds time.Duration
}

func (i increaseBlockTimeBy) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * i.seconds))

	return ctx, nil, true
}

func IncreaseBlockTimeBy(seconds time.Duration) Action {
	return increaseBlockTimeBy{seconds: seconds}
}

type setBlockTime struct {
	blockTime time.Time
}

func (s setBlockTime) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	return ctx.WithBlockTime(s.blockTime), nil, true
}

// SetBlockTime sets the block time to the given value
func SetBlockTime(blockTime time.Time) Action {
	return setBlockTime{blockTime: blockTime}
}

type setBlockNumber struct {
	blockNumber int64
}

func (s setBlockNumber) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	return ctx.WithBlockHeight(s.blockNumber), nil, true
}

// SetBlockNumber sets the block number to the given value
func SetBlockNumber(blockNumber int64) Action {
	return setBlockNumber{blockNumber: blockNumber}
}

type moveToNextBlock struct{}

func (m moveToNextBlock) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	app.EndBlock(types.RequestEndBlock{})
	app.Commit()

	newHeader := tmproto.Header{
		Height: ctx.BlockHeight() + 1,
		Time:   ctx.BlockTime().Add(time.Second * 5),
	}

	app.BeginBlock(types.RequestBeginBlock{
		Header: newHeader,
	})

	return app.NewContext(
		false,
		newHeader,
	).WithBlockTime(newHeader.Time), nil, true
}

func MoveToNextBlock() Action {
	return moveToNextBlock{}
}

type moveToNextBlockWithDuration struct {
	blockDuration time.Duration
}

func (m moveToNextBlockWithDuration) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	app.EndBlock(types.RequestEndBlock{Height: ctx.BlockHeight()})
	app.Commit()

	newHeader := tmproto.Header{
		Height: ctx.BlockHeight() + 1,
		Time:   ctx.BlockTime().Add(m.blockDuration),
	}

	app.BeginBlock(types.RequestBeginBlock{
		Header: newHeader,
	})

	return app.NewContext(
		false,
		newHeader,
	).WithBlockTime(newHeader.Time), nil, true
}

func MoveToNextBlockWithDuration(blockDuration time.Duration) Action {
	return moveToNextBlockWithDuration{
		blockDuration: blockDuration,
	}
}

type moveToNextBlockWithTime struct {
	blockTime time.Time
}

func (m moveToNextBlockWithTime) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	app.EndBlock(types.RequestEndBlock{Height: ctx.BlockHeight()})
	app.Commit()

	newHeader := tmproto.Header{
		Height: ctx.BlockHeight() + 1,
		Time:   m.blockTime,
	}

	app.BeginBlock(types.RequestBeginBlock{
		Header: newHeader,
	})

	return app.NewContext(
		false,
		newHeader,
	).WithBlockTime(newHeader.Time), nil, true
}

func MoveToNextBlockWithTime(blockTime time.Time) Action {
	return moveToNextBlockWithTime{
		blockTime: blockTime,
	}
}

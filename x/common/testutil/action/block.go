package action

import (
	"time"

	"github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

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

func (i increaseBlockTimeBy) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * i.seconds))

	return ctx, nil
}

func IncreaseBlockTimeBy(seconds time.Duration) Action {
	return increaseBlockTimeBy{seconds: seconds}
}

type setBlockTime struct {
	blockTime time.Time
}

func (s setBlockTime) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
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
	).WithBlockTime(newHeader.Time), nil
}

func MoveToNextBlock() Action {
	return moveToNextBlock{}
}

type moveToNextBlockWithDuration struct {
	blockDuration time.Duration
}

func (m moveToNextBlockWithDuration) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
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
	).WithBlockTime(newHeader.Time), nil
}

func MoveToNextBlockWithDuration(blockDuration time.Duration) Action {
	return moveToNextBlockWithDuration{
		blockDuration: blockDuration,
	}
}

type moveToNextBlockWithTime struct {
	blockTime time.Time
}

func (m moveToNextBlockWithTime) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
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
	).WithBlockTime(newHeader.Time), nil
}

func MoveToNextBlockWithTime(blockTime time.Time) Action {
	return moveToNextBlockWithTime{
		blockTime: blockTime,
	}
}

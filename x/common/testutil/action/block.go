package action

import (
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
)

type increaseBlockNumberBy struct {
	numBlocks int64
}

func (i increaseBlockNumberBy) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	app.EndBlocker(ctx)

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
	app.EndBlocker(ctx)
	app.Commit()

	newHeader := tmproto.Header{
		Height: ctx.BlockHeight() + 1,
		Time:   ctx.BlockTime().Add(time.Second * 5),
	}

	app.BeginBlocker(ctx)

	return app.NewContext(
		false,
	).WithBlockTime(newHeader.Time), nil
}

func MoveToNextBlock() Action {
	return moveToNextBlock{}
}

type moveToNextBlockWithDuration struct {
	blockDuration time.Duration
}

func (m moveToNextBlockWithDuration) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	app.EndBlocker(ctx)
	app.Commit()

	newHeader := tmproto.Header{
		Height: ctx.BlockHeight() + 1,
		Time:   ctx.BlockTime().Add(m.blockDuration),
	}

	app.BeginBlocker(ctx)

	return app.NewContext(
		false,
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
	app.EndBlocker(ctx)
	app.Commit()

	newHeader := tmproto.Header{
		Height: ctx.BlockHeight() + 1,
		Time:   m.blockTime,
	}

	app.BeginBlocker(ctx)

	return app.NewContext(
		false,
	).WithBlockTime(newHeader.Time), nil
}

func MoveToNextBlockWithTime(blockTime time.Time) Action {
	return moveToNextBlockWithTime{
		blockTime: blockTime,
	}
}

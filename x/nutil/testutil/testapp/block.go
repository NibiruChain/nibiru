package testapp

import (
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
)

// IncreaseBlockNumberBy runs EndBlocker for the current height and returns a
// context advanced by numBlocks.
func IncreaseBlockNumberBy(
	nibiru *app.NibiruApp, ctx sdk.Context, numBlocks int64,
) sdk.Context {
	nibiru.EndBlocker(ctx, abci.RequestEndBlock{Height: ctx.BlockHeight()})
	return ctx.WithBlockHeight(ctx.BlockHeight() + numBlocks)
}

// IncreaseBlockTimeBy returns a context advanced by blockDuration.
func IncreaseBlockTimeBy(ctx sdk.Context, blockDuration time.Duration) sdk.Context {
	return ctx.WithBlockTime(ctx.BlockTime().Add(blockDuration))
}

// SetBlockTime returns a context with blockTime as the current block time.
func SetBlockTime(ctx sdk.Context, blockTime time.Time) sdk.Context {
	return ctx.WithBlockTime(blockTime)
}

// SetBlockNumber returns a context with blockNumber as the current block height.
func SetBlockNumber(ctx sdk.Context, blockNumber int64) sdk.Context {
	return ctx.WithBlockHeight(blockNumber)
}

// MoveToNextBlock advances the app by one block using the historical default
// five-second block duration from the old action helper.
func MoveToNextBlock(nibiru *app.NibiruApp, ctx sdk.Context) sdk.Context {
	return MoveToNextBlockWithDuration(nibiru, ctx, 5*time.Second)
}

// MoveToNextBlockWithDuration commits the current block, begins the next block,
// and returns a fresh context for the new block.
func MoveToNextBlockWithDuration(
	nibiru *app.NibiruApp, ctx sdk.Context, blockDuration time.Duration,
) sdk.Context {
	return MoveToNextBlockWithTime(
		nibiru,
		ctx,
		ctx.BlockTime().Add(blockDuration),
	)
}

// MoveToNextBlockWithTime commits the current block, begins the next block at
// blockTime, and returns a fresh context for the new block.
func MoveToNextBlockWithTime(
	nibiru *app.NibiruApp, ctx sdk.Context, blockTime time.Time,
) sdk.Context {
	nibiru.EndBlock(abci.RequestEndBlock{Height: ctx.BlockHeight()})
	nibiru.Commit()

	newHeader := tmproto.Header{
		Height: ctx.BlockHeight() + 1,
		Time:   blockTime,
	}

	nibiru.BeginBlock(abci.RequestBeginBlock{Header: newHeader})

	return nibiru.NewContext(false, newHeader).WithBlockTime(newHeader.Time)
}

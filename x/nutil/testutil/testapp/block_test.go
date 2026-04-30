package testapp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBlockContextHelpers(t *testing.T) {
	nibiru, ctx := NewNibiruTestAppAndContext()

	startHeight := ctx.BlockHeight()
	startTime := ctx.BlockTime()
	nextTime := startTime.Add(11 * time.Second)

	ctx = IncreaseBlockNumberBy(nibiru, ctx, 3)
	require.Equal(t, startHeight+3, ctx.BlockHeight())

	ctx = IncreaseBlockTimeBy(ctx, 7*time.Second)
	require.Equal(t, startTime.Add(7*time.Second), ctx.BlockTime())

	ctx = SetBlockTime(ctx, nextTime)
	require.Equal(t, nextTime, ctx.BlockTime())

	ctx = SetBlockNumber(ctx, 42)
	require.Equal(t, int64(42), ctx.BlockHeight())
}

func TestMoveToNextBlock(t *testing.T) {
	nibiru, ctx := NewNibiruTestAppAndContext()

	startHeight := ctx.BlockHeight()
	startTime := ctx.BlockTime()

	ctx = MoveToNextBlock(nibiru, ctx)

	require.Equal(t, startHeight+1, ctx.BlockHeight())
	require.Equal(t, startTime.Add(5*time.Second), ctx.BlockTime())
}

func TestMoveToNextBlockWithDuration(t *testing.T) {
	nibiru, ctx := NewNibiruTestAppAndContext()

	startHeight := ctx.BlockHeight()
	startTime := ctx.BlockTime()

	ctx = MoveToNextBlockWithDuration(nibiru, ctx, 13*time.Second)

	require.Equal(t, startHeight+1, ctx.BlockHeight())
	require.Equal(t, startTime.Add(13*time.Second), ctx.BlockTime())
}

func TestMoveToNextBlockWithTime(t *testing.T) {
	nibiru, ctx := NewNibiruTestAppAndContext()

	startHeight := ctx.BlockHeight()
	nextTime := ctx.BlockTime().Add(time.Minute)

	ctx = MoveToNextBlockWithTime(nibiru, ctx, nextTime)

	require.Equal(t, startHeight+1, ctx.BlockHeight())
	require.Equal(t, nextTime, ctx.BlockTime())
}

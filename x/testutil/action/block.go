package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/testutil"
)

type increaseBlockNumberBy struct {
	numBlocks int64
}

func (i increaseBlockNumberBy) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
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

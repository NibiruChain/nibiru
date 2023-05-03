package action

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type startEpoch struct {
	epochIdentifier string
}

func (s startEpoch) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	epochInfo := app.EpochsKeeper.GetEpochInfo(ctx, s.epochIdentifier)
	epochInfo.EpochCountingStarted = true
	epochInfo.CurrentEpoch = 1
	epochInfo.CurrentEpochStartHeight = ctx.BlockHeight()
	epochInfo.CurrentEpochStartTime = ctx.BlockTime()
	epochInfo.StartTime = ctx.BlockTime()

	app.EpochsKeeper.UpsertEpochInfo(ctx, epochInfo)

	return ctx, nil, false
}

func StartEpoch(epochIdentifier string) action.Action {
	return startEpoch{epochIdentifier: epochIdentifier}
}

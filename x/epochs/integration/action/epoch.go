package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type startEpoch struct {
	epochIdentifier string
}

func (s startEpoch) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	epochInfo, err := app.EpochsKeeper.GetEpochInfo(ctx, s.epochIdentifier)
	if err != nil {
		return ctx, err, false
	}
	epochInfo.EpochCountingStarted = true
	epochInfo.CurrentEpoch = 1
	epochInfo.CurrentEpochStartHeight = ctx.BlockHeight()
	epochInfo.CurrentEpochStartTime = ctx.BlockTime()
	epochInfo.StartTime = ctx.BlockTime()

	app.EpochsKeeper.Epochs.Insert(ctx, epochInfo.Identifier, epochInfo)

	return ctx, nil, false
}

func StartEpoch(epochIdentifier string) action.Action {
	return startEpoch{epochIdentifier: epochIdentifier}
}

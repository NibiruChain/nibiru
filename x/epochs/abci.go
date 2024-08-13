package epochs

import (
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/epochs/keeper"
	"github.com/NibiruChain/nibiru/v2/x/epochs/types"
)

// BeginBlocker of epochs module.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	k.IterateEpochInfo(ctx, func(index int64, epochInfo types.EpochInfo) (stop bool) {
		if ctx.BlockTime().Before(epochInfo.StartTime) {
			return false
		}

		if !shouldEpochStart(epochInfo, ctx) {
			return false
		}

		epochInfo.CurrentEpochStartHeight = ctx.BlockHeight()
		epochInfo.CurrentEpochStartTime = ctx.BlockTime()

		if !epochInfo.EpochCountingStarted {
			epochInfo.EpochCountingStarted = true
			epochInfo.CurrentEpoch = 1
		} else {
			_ = ctx.EventManager().EmitTypedEvent(&types.EventEpochEnd{EpochNumber: epochInfo.CurrentEpoch})
			k.AfterEpochEnd(ctx, epochInfo.Identifier, epochInfo.CurrentEpoch)
			epochInfo.CurrentEpoch += 1
		}

		// emit new epoch start event, set epoch info, and run BeforeEpochStart hook
		_ = ctx.EventManager().EmitTypedEvent(&types.EventEpochStart{
			EpochNumber:    epochInfo.CurrentEpoch,
			EpochStartTime: epochInfo.CurrentEpochStartTime,
		})

		k.Epochs.Insert(ctx, epochInfo.Identifier, epochInfo)
		k.BeforeEpochStart(ctx, epochInfo.Identifier, epochInfo.CurrentEpoch)

		return false
	})
}

// shouldEpochStart checks if the epoch should start.
// an epoch is ready to start if:
// - it has not yet been initialized.
// - the current epoch end time is before the current block time
func shouldEpochStart(epochInfo types.EpochInfo, ctx sdk.Context) bool {
	// Epoch has not started yet
	if !epochInfo.EpochCountingStarted {
		return true
	}

	epochEndTime := epochInfo.CurrentEpochStartTime.Add(epochInfo.Duration)

	return ctx.BlockTime().After(epochEndTime) || ctx.BlockTime().Equal(epochEndTime)
}

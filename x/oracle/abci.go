package oracle

import (
	"time"

	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	"github.com/NibiruChain/nibiru/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker is called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	params, err := k.Params.Get(ctx)
	if err != nil {
		return
	}
	if types.IsPeriodLastBlock(ctx, params.VotePeriod) {
		k.UpdateExchangeRates(ctx)
	}

	// Do slash who did miss voting over threshold and
	// reset miss counters of all validators at the last block of slash window
	if types.IsPeriodLastBlock(ctx, params.SlashWindow) {
		k.SlashAndResetMissCounters(ctx)
	}

	// Clean up expired prices
	k.CleanExpiredRates(ctx)
}

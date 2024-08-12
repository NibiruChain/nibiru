package keeper

import (
	"fmt"

	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	epochstypes "github.com/NibiruChain/nibiru/v2/x/epochs/types"
	"github.com/NibiruChain/nibiru/v2/x/inflation/types"
)

// Hooks implements module-specific calls ([epochstypes.EpochHooks]) that will
// occur at the end of every epoch. Hooks is meant for use with
// `EpochsKeeper.SetHooks`. These functions run outside the normal body of
// transactions.
type Hooks struct {
	K Keeper
}

var _ epochstypes.EpochHooks = Hooks{}

// Hooks implements module-speecific calls that will occur in the ABCI
// BeginBlock logic.
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// BeforeEpochStart is a hook that runs just prior to the start of a new epoch.
func (h Hooks) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	// Perform no operations; we don't need to do anything here
	_, _, _ = ctx, epochIdentifier, epochNumber
}

// AfterEpochEnd is a hook that runs just prior to the first block whose
// timestamp is after the end of an epoch duration.
// AfterEpochEnd mints and allocates coins at the end of each epoch.
// If inflation is disabled as a module parameter, the state for
// "NumSkippedEpochs" increments.
func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	if epochIdentifier != epochstypes.DayEpochID {
		return
	}

	params := h.K.GetParams(ctx)

	// Skip inflation if it is disabled and increment number of skipped epochs
	if !params.InflationEnabled {
		var prevSkippedEpochs uint64
		if !params.HasInflationStarted {
			// If the inflation never started, we use epochNumber as the number of skipped epochs
			// to avoid missing periods when we upgrade a chain.
			h.K.NumSkippedEpochs.Set(ctx, epochNumber)
			prevSkippedEpochs = epochNumber
		} else {
			prevSkippedEpochs = h.K.NumSkippedEpochs.Next(ctx)
		}

		h.K.Logger(ctx).Debug(
			"skipping inflation mint and allocation",
			"height", ctx.BlockHeight(),
			"epoch-id", epochIdentifier,
			"epoch-number", epochNumber,
			"skipped-epochs", prevSkippedEpochs,
		)
		return
	}

	// mint coins, update supply
	period := h.K.CurrentPeriod.Peek(ctx)
	epochsPerPeriod := h.K.GetEpochsPerPeriod(ctx)

	epochMintProvision := types.CalculateEpochMintProvision(
		params,
		period,
	)

	if !epochMintProvision.IsPositive() {
		h.K.Logger(ctx).Error(
			"SKIPPING INFLATION: negative epoch mint provision",
			"value", epochMintProvision.String(),
		)
		return
	}

	mintedCoin := sdk.Coin{
		Denom:  denoms.NIBI,
		Amount: epochMintProvision.TruncateInt(),
	}

	staking, strategic, communityPool, err := h.K.MintAndAllocateInflation(ctx, mintedCoin, params)
	if err != nil {
		h.K.Logger(ctx).Error(
			"SKIPPING INFLATION: failed to mint and allocate inflation",
			"error", err,
		)
		return
	}

	// If period is passed, update the period. A period is
	// passed if the current epoch number surpasses the epochsPerPeriod for the
	// current period. Skipped epochs are subtracted to only account for epochs
	// where inflation minted tokens.
	//
	// Examples:
	//  Given, epochNumber = 1, period = 0, epochPerPeriod = 30, skippedEpochs = 0
	//    => 1 - 30 * 0 - 0 < 30
	//    => nothing to do here
	//  Given, epochNumber = 70, period = 1, epochPerPeriod = 30, skippedEpochs = 10
	//    => 70 - 1 * 30 - 10 >= 30
	//    => a period has ended! we set a new period
	//  Given, epochNumber = 42099, period = 0, epochPerPeriod = 30, skippedEpochs = 42069
	//    => 42099 - 0 * 30 - 42069 >= 30
	//    => a period has ended! we set a new period
	numSkippedEpochs := h.K.NumSkippedEpochs.Peek(ctx)
	if int64(epochNumber)-
		int64(epochsPerPeriod*period)-
		int64(numSkippedEpochs) >= int64(epochsPerPeriod) {
		periodBeforeIncrement := h.K.CurrentPeriod.Next(ctx)

		h.K.Logger(ctx).Info(fmt.Sprintf("setting new period: %d", periodBeforeIncrement+1))
	}

	defer func() {
		stakingAmt := staking.Amount
		strategicAmt := strategic.Amount
		cpAmt := communityPool.Amount

		if mintedCoin.Amount.IsInt64() {
			telemetry.IncrCounterWithLabels(
				[]string{types.ModuleName, "allocate", "total"},
				float32(mintedCoin.Amount.Int64()),
				[]metrics.Label{telemetry.NewLabel("denom", mintedCoin.Denom)},
			)
		}
		if stakingAmt.IsInt64() {
			telemetry.IncrCounterWithLabels(
				[]string{types.ModuleName, "allocate", "staking", "total"},
				float32(stakingAmt.Int64()),
				[]metrics.Label{telemetry.NewLabel("denom", mintedCoin.Denom)},
			)
		}
		if strategicAmt.IsInt64() {
			telemetry.IncrCounterWithLabels(
				[]string{types.ModuleName, "allocate", "strategic", "total"},
				float32(strategicAmt.Int64()),
				[]metrics.Label{telemetry.NewLabel("denom", mintedCoin.Denom)},
			)
		}
		if cpAmt.IsInt64() {
			telemetry.IncrCounterWithLabels(
				[]string{types.ModuleName, "allocate", "community_pool", "total"},
				float32(cpAmt.Int64()),
				[]metrics.Label{telemetry.NewLabel("denom", mintedCoin.Denom)},
			)
		}
	}()

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeInflation,
			sdk.NewAttribute(types.AttributeEpochNumber, fmt.Sprintf("%d", epochNumber-numSkippedEpochs)),
			sdk.NewAttribute(types.AttributeKeyEpochProvisions, epochMintProvision.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, mintedCoin.Amount.String()),
		),
	)
}

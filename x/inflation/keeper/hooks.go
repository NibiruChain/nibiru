package keeper

import (
	"fmt"

	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/inflation/types"
)

// BeforeEpochStart: noop, We don't need to do anything here
func (k Keeper) BeforeEpochStart(_ sdk.Context, _ string, _ uint64) {}

// AfterEpochEnd mints and allocates coins at the end of each epoch end
func (k Keeper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	if epochIdentifier != epochstypes.DayEpochID {
		return
	}

	params := k.GetParams(ctx)

	// Skip inflation if it is disabled and increment number of skipped epochs
	if !params.InflationEnabled {
		prevSkippedEpochs := k.NumSkippedEpochs.Next(ctx)

		k.Logger(ctx).Debug(
			"skipping inflation mint and allocation",
			"height", ctx.BlockHeight(),
			"epoch-id", epochIdentifier,
			"epoch-number", epochNumber,
			"skipped-epochs", prevSkippedEpochs+1,
		)
		return
	}

	// mint coins, update supply
	period := k.CurrentPeriod.Peek(ctx)
	epochsPerPeriod := k.EpochsPerPeriod(ctx)

	epochMintProvision := types.CalculateEpochMintProvision(
		params,
		period,
	)

	if !epochMintProvision.IsPositive() {
		k.Logger(ctx).Error(
			"SKIPPING INFLATION: negative epoch mint provision",
			"value", epochMintProvision.String(),
		)
		return
	}

	mintedCoin := sdk.Coin{
		Denom:  denoms.NIBI,
		Amount: epochMintProvision.TruncateInt(),
	}

	staking, incentives, communityPool, err := k.MintAndAllocateInflation(ctx, mintedCoin, params)
	if err != nil {
		panic(err)
	}

	// If period is passed, update the period. A period is
	// passed if the current epoch number surpasses the epochsPerPeriod for the
	// current period. Skipped epochs are subtracted to only account for epochs
	// where inflation minted tokens.
	//
	// Examples:
	// Given, epochNumber = 1, period = 0, epochPerPeriod = 365, skippedEpochs = 0
	//   => 1 - 365 * 0 - 0 < 365 --- nothing to do here
	// Given, epochNumber = 741, period = 1, epochPerPeriod = 365, skippedEpochs = 10
	//   => 741 - 1 * 365 - 10 > 365 --- a period has passed! we set a new period
	if int64(epochNumber)-
		int64(epochsPerPeriod*period)-
		int64(k.NumSkippedEpochs.Peek(ctx)) > int64(epochsPerPeriod) {
		k.CurrentPeriod.Next(ctx)
	}

	defer func() {
		stakingAmt := staking.Amount
		incentivesAmt := incentives.Amount
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
		if incentivesAmt.IsInt64() {
			telemetry.IncrCounterWithLabels(
				[]string{types.ModuleName, "allocate", "incentives", "total"},
				float32(incentivesAmt.Int64()),
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
			types.EventTypeMint,
			sdk.NewAttribute(types.AttributeEpochNumber, fmt.Sprintf("%d", epochNumber)),
			sdk.NewAttribute(types.AttributeKeyEpochProvisions, epochMintProvision.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, mintedCoin.Amount.String()),
		),
	)
}

// ___________________________________________________________________________________________________

// Hooks wrapper struct for incentives keeper
type Hooks struct {
	k Keeper
}

var _ epochstypes.EpochHooks = Hooks{}

// Return the wrapper struct
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// epochs hooks
func (h Hooks) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	h.k.BeforeEpochStart(ctx, epochIdentifier, epochNumber)
}

func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	h.k.AfterEpochEnd(ctx, epochIdentifier, epochNumber)
}

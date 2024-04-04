package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"
)

// SlashAndResetMissCounters do slash any operator who over criteria & clear all operators miss counter to zero
func (k Keeper) SlashAndResetMissCounters(ctx sdk.Context) {
	height := ctx.BlockHeight()
	distributionHeight := height - sdk.ValidatorUpdateDelay - 1

	// slash_window / vote_period
	votePeriodsPerWindow := uint64(
		sdkmath.LegacyNewDec(int64(k.SlashWindow(ctx))).
			QuoInt64(int64(k.VotePeriod(ctx))).
			TruncateInt64(),
	)
	minValidPerWindow := k.MinValidPerWindow(ctx)
	slashFraction := k.SlashFraction(ctx)
	powerReduction := k.StakingKeeper.PowerReduction(ctx)

	iter, err := k.MissCounters.Iterate(ctx, &collections.Range[sdk.ValAddress]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate miss counter", "error", err)
		return
	}
	kv, err := iter.KeyValues()
	if err != nil {
		k.Logger(ctx).Error("failed to get miss counter key values", "error", err)
		return
	}

	for _, mc := range kv {
		operator := mc.Key
		missCounter := mc.Value
		// Calculate valid vote rate; (SlashWindow - MissCounter)/SlashWindow
		validVoteRate := sdkmath.LegacyNewDecFromInt(
			sdkmath.NewInt(int64(votePeriodsPerWindow - missCounter))).
			QuoInt64(int64(votePeriodsPerWindow))

		// Penalize the validator whose the valid vote rate is smaller than min threshold
		if validVoteRate.LT(minValidPerWindow) {
			validator, err := k.StakingKeeper.Validator(ctx, operator)
			if err != nil {
				k.Logger(ctx).Error("failed getting staking keeper validator", "error", err)
				continue
			}

			if validator.IsBonded() && !validator.IsJailed() {
				consAddr, err := validator.GetConsAddr()
				if err != nil {
					k.Logger(ctx).Error("fail to get consensus address", "validator", validator.GetOperator())
					continue
				}

				k.StakingKeeper.Slash(
					ctx, consAddr,
					distributionHeight, validator.GetConsensusPower(powerReduction), slashFraction,
				)
				k.Logger(ctx).Info("slash", "validator", consAddr, "fraction", slashFraction.String())
				k.StakingKeeper.Jail(ctx, consAddr)
			}
		}

		err := k.MissCounters.Remove(ctx, operator)
		if err != nil {
			k.Logger(ctx).Error("fail to delete miss counter", "operator", operator.String(), "error", err)
		}
	}
}

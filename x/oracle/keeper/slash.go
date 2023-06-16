package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
)

// SlashAndResetMissCounters do slash any operator who over criteria & clear all operators miss counter to zero
func (k Keeper) SlashAndResetMissCounters(ctx sdk.Context) {
	height := ctx.BlockHeight()
	distributionHeight := height - sdk.ValidatorUpdateDelay - 1

	// slash_window / vote_period
	votePeriodsPerWindow := uint64(
		sdk.NewDec(int64(k.SlashWindow(ctx))).
			QuoInt64(int64(k.VotePeriod(ctx))).
			TruncateInt64(),
	)
	minValidPerWindow := k.MinValidPerWindow(ctx)
	slashFraction := k.SlashFraction(ctx)
	powerReduction := k.StakingKeeper.PowerReduction(ctx)

	for _, mc := range k.MissCounters.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
		operator := mc.Key
		missCounter := mc.Value
		// Calculate valid vote rate; (SlashWindow - MissCounter)/SlashWindow
		validVoteRate := sdk.NewDecFromInt(
			sdk.NewInt(int64(votePeriodsPerWindow - missCounter))).
			QuoInt64(int64(votePeriodsPerWindow))

		// Penalize the validator whose the valid vote rate is smaller than min threshold
		if validVoteRate.LT(minValidPerWindow) {
			validator := k.StakingKeeper.Validator(ctx, operator)
			if validator.IsBonded() && !validator.IsJailed() {
				consAddr, err := validator.GetConsAddr()
				if err != nil {
					k.Logger(ctx).Error("fail to get consensus address", "validator", validator.GetOperator().String())
				}

				k.StakingKeeper.Slash(
					ctx, consAddr,
					distributionHeight, validator.GetConsensusPower(powerReduction), slashFraction,
				)
				k.Logger(ctx).Info("slash", "validator", consAddr.String(), "fraction", slashFraction.String())
				k.StakingKeeper.Jail(ctx, consAddr)
			}
		}

		err := k.MissCounters.Delete(ctx, operator)
		if err != nil {
			k.Logger(ctx).Error("fail to delete miss counter", "operator", operator.String(), "error", err)
		}
	}
}

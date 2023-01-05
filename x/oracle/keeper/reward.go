package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// AllocatePairRewards allocates rewards for the current pair.
// TODO: This function is still not in use.
func (k Keeper) AllocatePairRewards(ctx sdk.Context, funderModule string, pair string, totalCoins sdk.Coins, votePeriods uint64) error {
	if !k.Pairs.Has(ctx, pair) {
		return types.ErrUnknownPair.Wrap(pair)
	}

	votePeriodCoins := make(sdk.Coins, len(totalCoins))
	for i, coin := range totalCoins {
		newCoin := sdk.NewCoin(coin.Denom, coin.Amount.QuoRaw(int64(votePeriods)))
		votePeriodCoins[i] = newCoin
	}

	id := k.PairRewardsID.Next(ctx)
	k.PairRewards.Insert(ctx, id, types.PairReward{
		Pair:        pair,
		Id:          id,
		VotePeriods: votePeriods,
		Coins:       votePeriodCoins,
	})

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, funderModule, types.ModuleName, totalCoins)
}

// rewardBallotWinners implements at the end of every VotePeriod,
// give out a portion of spread fees collected in the oracle reward pool
// to the oracle voters that voted faithfully.
func (k Keeper) rewardBallotWinners(
	ctx sdk.Context,
	whitelistedPairs map[string]struct{},
	validatorPerformanceMap map[string]types.ValidatorPerformance,
) {
	validatorsWeightSum := types.GetValidatorWeightSum(validatorPerformanceMap)
	if validatorsWeightSum == 0 {
		return
	}

	var periodRewards sdk.DecCoins
	for pair := range whitelistedPairs {
		rewardsForPair := k.AccrueVotePeriodPairRewards(ctx, pair)

		// return if there's no rewards to give out
		if rewardsForPair.IsZero() {
			continue
		}

		periodRewards = periodRewards.Add(sdk.NewDecCoinsFromCoins(rewardsForPair...)...)
	}

	// Dole out rewards
	var distributedReward sdk.Coins
	for _, winner := range validatorPerformanceMap {
		receiverVal := k.StakingKeeper.Validator(ctx, winner.ValAddress)

		// Reflects contribution
		rewardCoins, _ := periodRewards.MulDec(sdk.NewDec(winner.Weight).QuoInt64(validatorsWeightSum)).TruncateDecimal()

		// In case absence of the validator, we just skip distribution
		if receiverVal != nil && !rewardCoins.IsZero() {
			k.distrKeeper.AllocateTokensToValidator(ctx, receiverVal, sdk.NewDecCoinsFromCoins(rewardCoins...))
			distributedReward = distributedReward.Add(rewardCoins...)
		}
	}

	// Move distributed reward to distribution module
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, k.distrName, distributedReward)
	if err != nil {
		panic(fmt.Sprintf("[oracle] Failed to send coins to distribution module %s", err.Error()))
	}
}

// AccrueVotePeriodPairRewards retrieves the vote period rewards for the provided pair.
// And decreases the distribution period count of each pair reward instance.
// If the distribution period count drops to 0: the reward instance is removed.
func (k Keeper) AccrueVotePeriodPairRewards(ctx sdk.Context, pair string) sdk.Coins {
	coins := sdk.NewCoins()

	for _, rewardID := range k.PairRewards.Indexes.RewardsByPair.ExactMatch(ctx, pair).PrimaryKeys() {
		r, err := k.PairRewards.Get(ctx, rewardID)
		if err != nil {
			panic(err)
		}
		// add coin rewards
		coins = coins.Add(r.Coins...)
		// update pair reward distribution count
		// if vote period == 0, then delete
		r.VotePeriods -= 1
		if r.VotePeriods == 0 {
			err := k.PairRewards.Delete(ctx, rewardID)
			if err != nil {
				panic(err)
			}
		} else {
			k.PairRewards.Insert(ctx, rewardID, r)
		}
	}

	return coins
}

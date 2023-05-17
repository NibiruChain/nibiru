package keeper

import (
	"fmt"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func (k Keeper) AllocateRewards(ctx sdk.Context, funderModule string, totalCoins sdk.Coins, votePeriods uint64) error {
	votePeriodCoins := make(sdk.Coins, len(totalCoins))
	for i, coin := range totalCoins {
		newCoin := sdk.NewCoin(coin.Denom, coin.Amount.QuoRaw(int64(votePeriods)))
		votePeriodCoins[i] = newCoin
	}

	id := k.RewardsID.Next(ctx)
	k.Rewards.Insert(ctx, id, types.Rewards{
		Id:          id,
		VotePeriods: votePeriods,
		Coins:       votePeriodCoins,
	})

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, funderModule, types.ModuleName, totalCoins)
}

// rewardBallotWinners gives out a portion of spread fees collected in the
// oracle reward pool to the oracle voters that voted faithfully.
func (k Keeper) rewardBallotWinners(
	ctx sdk.Context,
	validatorPerformances types.ValidatorPerformances,
) {
	totalRewardWeight := validatorPerformances.GetTotalRewardWeight()
	if totalRewardWeight == 0 {
		return
	}

	var totalRewards sdk.DecCoins
	rewards := k.GatherRewardsForVotePeriod(ctx)
	totalRewards = totalRewards.Add(sdk.NewDecCoinsFromCoins(rewards...)...)

	var distributedRewards sdk.Coins
	for _, validatorPerformance := range validatorPerformances {
		validator := k.StakingKeeper.Validator(ctx, validatorPerformance.ValAddress)
		if validator == nil {
			continue
		}

		rewardPortion, _ := totalRewards.MulDec(sdk.NewDec(validatorPerformance.RewardWeight).QuoInt64(totalRewardWeight)).TruncateDecimal()
		k.distrKeeper.AllocateTokensToValidator(ctx, validator, sdk.NewDecCoinsFromCoins(rewardPortion...))
		distributedRewards = distributedRewards.Add(rewardPortion...)
	}

	// Move distributed reward to distribution module
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, k.distrModuleName, distributedRewards)
	if err != nil {
		panic(fmt.Sprintf("[oracle] Failed to send coins to distribution module %s", err.Error()))
	}
}

// GatherRewardsForVotePeriod retrieves the pair rewards for the provided pair and current vote period.
func (k Keeper) GatherRewardsForVotePeriod(ctx sdk.Context) sdk.Coins {
	coins := sdk.NewCoins()
	// iterate over
	for _, rewardId := range k.Rewards.Iterate(ctx, collections.Range[uint64]{}).Keys() {
		pairReward, err := k.Rewards.Get(ctx, rewardId)
		if err != nil {
			panic(fmt.Sprintf("[oracle] Failed to get reward %s", err.Error()))
		}
		coins = coins.Add(pairReward.Coins...)

		// Decrease the remaining vote periods of the PairReward.
		pairReward.VotePeriods -= 1
		if pairReward.VotePeriods == 0 {
			// If the distribution period count drops to 0: the reward instance is removed.
			err := k.Rewards.Delete(ctx, rewardId)
			if err != nil {
				panic(fmt.Sprintf("[oracle] Failed to delete pair reward %s", err.Error()))
			}
		} else {
			k.Rewards.Insert(ctx, rewardId, pairReward)
		}
	}

	return coins
}

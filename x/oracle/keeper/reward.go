package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func (k Keeper) AllocatePairRewards(ctx sdk.Context, funderModule string, pair common.AssetPair, totalCoins sdk.Coins, votePeriods uint64) error {
	if !k.WhitelistedPairs.Has(ctx, pair) {
		return types.ErrUnknownPair.Wrap(pair.String())
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

// rewardBallotWinners gives out a portion of spread fees collected in the
// oracle reward pool to the oracle voters that voted faithfully.
func (k Keeper) rewardBallotWinners(
	ctx sdk.Context,
	whitelistedPairs map[common.AssetPair]struct{},
	validatorPerformanceMap map[string]types.ValidatorPerformance,
) {
	totalRewardWeight := types.GetTotalRewardWeight(validatorPerformanceMap)
	if totalRewardWeight == 0 {
		return
	}

	var totalRewards sdk.DecCoins
	for pair := range whitelistedPairs {
		pairRewards := k.GatherRewardsForVotePeriod(ctx, pair)

		if pairRewards.IsZero() {
			continue
		}

		totalRewards = totalRewards.Add(sdk.NewDecCoinsFromCoins(pairRewards...)...)
	}

	// Dole out rewards
	var distributedRewards sdk.Coins
	for _, validatorPerformance := range validatorPerformanceMap {
		validator := k.StakingKeeper.Validator(ctx, validatorPerformance.ValAddress)
		if validator == nil {
			continue
		}

		// Reflects contribution
		rewardPortion, _ := totalRewards.MulDec(sdk.NewDec(validatorPerformance.RewardWeight).QuoInt64(totalRewardWeight)).TruncateDecimal()
		k.distrKeeper.AllocateTokensToValidator(ctx, validator, sdk.NewDecCoinsFromCoins(rewardPortion...))
		distributedRewards = distributedRewards.Add(rewardPortion...)
	}

	// Move distributed reward to distribution module
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, k.distrModuleName, distributedRewards)
	if err != nil {
		// TODO(k-yang): revisit panic behavior
		panic(fmt.Sprintf("[oracle] Failed to send coins to distribution module %s", err.Error()))
	}
}

// GatherRewardsForVotePeriod retrieves the pair rewards for the provided pair and current vote period.
func (k Keeper) GatherRewardsForVotePeriod(ctx sdk.Context, pair common.AssetPair) sdk.Coins {
	coins := sdk.NewCoins()
	// iterate over
	for _, rewardId := range k.PairRewards.Indexes.RewardsByPair.ExactMatch(ctx, pair).PrimaryKeys() {
		pairReward, err := k.PairRewards.Get(ctx, rewardId)
		if err != nil {
			// TODO(k-yang): revisit panic behavior
			panic(err)
		}
		coins = coins.Add(pairReward.Coins...)

		// Decrease the remaining vote periods of the PairRward.
		pairReward.VotePeriods -= 1
		if pairReward.VotePeriods == 0 {
			// If the distribution period count drops to 0: the reward instance is removed.
			err := k.PairRewards.Delete(ctx, rewardId)
			if err != nil {
				// TODO(k-yang): revisit panic behavior
				panic(err)
			}
		} else {
			k.PairRewards.Insert(ctx, rewardId, pairReward)
		}
	}

	return coins
}

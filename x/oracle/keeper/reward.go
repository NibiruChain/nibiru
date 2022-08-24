package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func (k Keeper) AllocatePairRewards(ctx sdk.Context, funderModule string, pair string, totalCoins sdk.Coins, votePeriods uint64) error {
	// check if pair exists
	if !k.PairExists(ctx, pair) {
		return types.ErrUnknownPair.Wrap(pair)
	}

	votePeriodCoins := make(sdk.Coins, len(totalCoins))
	for i, coin := range totalCoins {
		newCoin := sdk.NewCoin(coin.Denom, coin.Amount.QuoRaw(int64(votePeriods)))
		votePeriodCoins[i] = newCoin
	}

	k.CreatePairReward(ctx, &types.PairReward{
		Pair:        pair,
		VotePeriods: votePeriods,
		Coins:       votePeriodCoins,
	})

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, funderModule, types.ModuleName, totalCoins)
}

func (k Keeper) CreatePairReward(ctx sdk.Context, rewards *types.PairReward) {
	rewards.Id = k.NextPairRewardKey(ctx)
	k.SetPairReward(ctx, rewards)
}

func (k Keeper) DeletePairReward(ctx sdk.Context, pair string, id uint64) error {
	pk := types.GetPairRewardsKey(pair, id)
	store := ctx.KVStore(k.storeKey)
	if !store.Has(pk) {
		return fmt.Errorf("unknown pair rewards key: %s %d", pair, id)
	}

	store.Delete(pk)
	return nil
}

func (k Keeper) IteratePairRewards(ctx sdk.Context, pair string, do func(rewards *types.PairReward) (stop bool)) {
	pfx := types.GetPairRewardsPrefixKey(pair)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), pfx)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		rewards := new(types.PairReward)
		k.cdc.MustUnmarshal(iter.Value(), rewards)
		if do(rewards) {
			break
		}
	}
}

func (k Keeper) IterateAllPairRewards(ctx sdk.Context, do func(rewards *types.PairReward) (stop bool)) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PairRewardsKey)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		rewards := new(types.PairReward)
		k.cdc.MustUnmarshal(iter.Value(), rewards)
		if do(rewards) {
			break
		}
	}
}

func (k Keeper) GetPairReward(ctx sdk.Context, pair string, id uint64) (*types.PairReward, error) {
	pk := types.GetPairRewardsKey(pair, id)
	v := ctx.KVStore(k.storeKey).Get(pk)
	if v == nil {
		return nil, fmt.Errorf("not found")
	}
	r := new(types.PairReward)
	k.cdc.MustUnmarshal(v, r)
	return r, nil
}

func (k Keeper) SetPairReward(ctx sdk.Context, rewards *types.PairReward) {
	pk := types.GetPairRewardsKey(rewards.Pair, rewards.Id)
	ctx.KVStore(k.storeKey).Set(pk, k.cdc.MustMarshal(rewards))
}

func (k Keeper) NextPairRewardKey(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	if v := store.Get(types.PairRewardsCounterKey); v != nil {
		id := sdk.BigEndianToUint64(v)
		store.Set(types.PairRewardsCounterKey, sdk.Uint64ToBigEndian(id+1))
		return id
	} else {
		store.Set(types.PairRewardsCounterKey, sdk.Uint64ToBigEndian(1))
		return 0
	}
}

// RewardBallotWinners implements at the end of every VotePeriod,
// give out a portion of spread fees collected in the oracle reward pool
// to the oracle voters that voted faithfully.
func (k Keeper) RewardBallotWinners(
	ctx sdk.Context,
	voteTargets map[string]struct{},
	ballotWinners map[string]types.ValidatorPerformance,
) {
	rewardPair := make([]string, len(voteTargets))

	i := 0
	for pair := range voteTargets {
		rewardPair[i] = pair
		i++
	}

	// Sum weight of the claims
	ballotPowerSum := int64(0)
	for _, winner := range ballotWinners {
		ballotPowerSum += winner.Weight
	}

	// Exit if the ballot is empty
	if ballotPowerSum == 0 {
		return
	}

	var periodRewards sdk.DecCoins
	for _, pair := range rewardPair {
		rewardsForPair := k.AccrueVotePeriodPairRewards(ctx, pair)

		// return if there's no rewards to give out
		if rewardsForPair.IsZero() {
			continue
		}

		periodRewards = periodRewards.Add(sdk.NewDecCoinsFromCoins(rewardsForPair...)...)
	}

	// Dole out rewards
	var distributedReward sdk.Coins
	for _, winner := range ballotWinners {
		receiverVal := k.StakingKeeper.Validator(ctx, winner.ValAddress)

		// Reflects contribution
		rewardCoins, _ := periodRewards.MulDec(sdk.NewDec(winner.Weight).QuoInt64(ballotPowerSum)).TruncateDecimal()

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
// TODO(mercilex): don't like API name
func (k Keeper) AccrueVotePeriodPairRewards(ctx sdk.Context, pair string) sdk.Coins {
	var pairRewards []*types.PairReward
	k.IteratePairRewards(ctx, pair, func(rewards *types.PairReward) (stop bool) {
		pairRewards = append(pairRewards, rewards)
		return false
	})

	coins := sdk.NewCoins()
	// iterate over
	for _, r := range pairRewards {
		// add coin rewards
		coins = coins.Add(r.Coins...)
		// update pair reward distribution count
		// if vote period == 0, then delete
		r.VotePeriods -= 1
		if r.VotePeriods == 0 {
			err := k.DeletePairReward(ctx, r.Pair, r.Id)
			if err != nil {
				panic(err)
			}
		} else {
			k.SetPairReward(ctx, r)
		}
	}

	return coins
}

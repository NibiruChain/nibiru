package keeper

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func (k Keeper) AllocateRewards(ctx sdk.Context, pair string, totalCoins sdk.Coins, votePeriods uint64) error {
	// check if pair exists
	if !k.PairExists(ctx, pair) {
		return types.ErrUnknownPair.Wrap(pair)
	}

	k.CreatePairReward(ctx, &types.PairReward{
		Pair:        pair,
		VotePeriods: votePeriods,
		Coins:       totalCoins,
	})

	return nil
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
	rewardDenoms := make([]string, len(voteTargets)+1)
	rewardDenoms[0] = common.DenomGov

	i := 1
	for denom := range voteTargets {
		rewardDenoms[i] = denom
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
	for _, denom := range rewardDenoms {
		rewardsForPair := k.GetRewardsForPair(ctx, denom)

		// return if there's no rewards to give out
		if rewardsForPair.IsZero() {
			continue
		}

		periodRewards = periodRewards.Add(sdk.NewDecCoinsFromCoins(rewardsForPair)...)
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

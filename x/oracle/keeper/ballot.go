package keeper

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/omap"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// groupVotesByPair takes a collection of votes and groups them by their
// associated asset pair. This method only considers votes from active validators
// and disregards votes from validators that are not in the provided validator set.
//
// Note that any abstain votes (votes with a non-positive exchange rate) are
// assigned zero vote power. This function then returns a map where each
// asset pair is associated with its collection of ExchangeRateVotes.
func (k Keeper) groupVotesByPair(
	ctx sdk.Context,
	validatorPerformances types.ValidatorPerformances,
) (pairVotes map[asset.Pair]types.ExchangeRateVotes) {
	pairVotes = map[asset.Pair]types.ExchangeRateVotes{}

	iter, err := k.Votes.Iterate(ctx, &collections.Range[sdk.ValAddress]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate votes", "error", err)
		return
	}
	kv, err := iter.KeyValues()
	if err != nil {
		k.Logger(ctx).Error("failed to get votes key values", "error", err)
		return
	}
	for _, value := range kv {
		voterAddr, aggregateVote := value.Key, value.Value

		// skip votes from inactive validators
		validatorPerformance, exists := validatorPerformances[aggregateVote.Voter]
		if !exists {
			continue
		}

		for _, tuple := range aggregateVote.ExchangeRateTuples {
			power := validatorPerformance.Power
			if !tuple.ExchangeRate.IsPositive() {
				// Make the power of abstain vote zero
				power = 0
			}

			pairVotes[tuple.Pair] = append(
				pairVotes[tuple.Pair],
				types.NewExchangeRateVote(
					tuple.ExchangeRate,
					tuple.Pair,
					voterAddr,
					power,
				),
			)
		}
	}

	return
}

// clearVotesAndPrevotes clears all tallied prevotes and votes from the store
func (k Keeper) clearVotesAndPrevotes(ctx sdk.Context, votePeriod uint64) {
	// Clear all aggregate prevotes
	iterPrevotes, err := k.Prevotes.Iterate(ctx, &collections.Range[sdk.ValAddress]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate prevotes", "error", err)
		return
	}
	kvPrevotes, err := iterPrevotes.KeyValues()
	if err != nil {
		k.Logger(ctx).Error("failed to get prevotes key values", "error", err)
		return
	}

	for _, prevote := range kvPrevotes {
		valAddr, aggregatePrevote := prevote.Key, prevote.Value
		if ctx.BlockHeight() >= int64(aggregatePrevote.SubmitBlock+votePeriod) {
			err := k.Prevotes.Remove(ctx, valAddr)
			if err != nil {
				k.Logger(ctx).Error("failed to delete prevote", "error", err)
			}
		}
	}

	// Clear all aggregate votes
	iterVotes, err := k.Votes.Iterate(ctx, &collections.Range[sdk.ValAddress]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate votes", "error", err)
		return
	}
	keyVotes, err := iterVotes.Keys()
	if err != nil {
		k.Logger(ctx).Error("failed to get votes keys", "error", err)
		return
	}
	for _, valAddr := range keyVotes {
		err := k.Votes.Remove(ctx, valAddr)
		if err != nil {
			k.Logger(ctx).Error("failed to delete vote", "error", err)
		}
	}
}

// isPassingVoteThreshold votes is passing the threshold amount of voting power
func isPassingVoteThreshold(
	votes types.ExchangeRateVotes, thresholdVotingPower sdkmath.Int, minVoters uint64,
) bool {
	totalPower := sdkmath.NewInt(votes.Power())
	if totalPower.IsZero() {
		return false
	}

	if totalPower.LT(thresholdVotingPower) {
		return false
	}

	if votes.NumValidVoters() < minVoters {
		return false
	}

	return true
}

// removeInvalidVotes removes the votes which have not reached the vote
// threshold or which are not part of the whitelisted pairs anymore: example
// when params change during a vote period but some votes were already made.
//
// ALERT: This function mutates the pairVotes map, it removes the votes for
// the pair which is not passing the threshold or which is not whitelisted
// anymore.
func (k Keeper) removeInvalidVotes(
	ctx sdk.Context,
	pairVotes map[asset.Pair]types.ExchangeRateVotes,
	whitelistedPairs set.Set[asset.Pair],
) {
	totalBondedTokens, err := k.StakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		k.Logger(ctx).Error("failed to get total bonded tokens", "error", err)
		return
	}

	totalBondedPower := sdk.TokensToConsensusPower(
		totalBondedTokens, k.StakingKeeper.PowerReduction(ctx),
	)

	// Iterate through sorted keys for deterministic ordering.
	orderedPairVotes := omap.OrderedMap_Pair[types.ExchangeRateVotes](pairVotes)
	for pair := range orderedPairVotes.Range() {
		// If pair is not whitelisted, or the votes for it has failed, then skip
		// and remove it from pairBallotsMap for iteration efficiency
		if !whitelistedPairs.Has(pair) {
			delete(pairVotes, pair)
		}

		// If the votes is not passed, remove it from the whitelistedPairs set
		// to prevent slashing validators who did valid vote.
		if !isPassingVoteThreshold(
			pairVotes[pair],
			k.VoteThreshold(ctx).MulInt64(totalBondedPower).RoundInt(),
			k.MinVoters(ctx),
		) {
			delete(whitelistedPairs, pair)
			delete(pairVotes, pair)
			continue
		}
	}
}

// Tally calculates the median and returns it. Sets the set of voters to be
// rewarded, i.e. voted within a reasonable spread from the weighted median to
// the store.
//
// ALERT: This function mutates validatorPerformances slice based on the votes
// made by the validators.
func Tally(
	votes types.ExchangeRateVotes,
	rewardBand sdkmath.LegacyDec,
	validatorPerformances types.ValidatorPerformances,
) sdkmath.LegacyDec {
	weightedMedian := votes.WeightedMedianWithAssertion()
	standardDeviation := votes.StandardDeviation(weightedMedian)
	rewardSpread := weightedMedian.Mul(rewardBand.QuoInt64(2))

	if standardDeviation.GT(rewardSpread) {
		rewardSpread = standardDeviation
	}

	for _, v := range votes {
		// Filter votes winners & abstain voters
		isInsideSpread := v.ExchangeRate.GTE(weightedMedian.Sub(rewardSpread)) &&
			v.ExchangeRate.LTE(weightedMedian.Add(rewardSpread))
		isAbstainVote := !v.ExchangeRate.IsPositive() // strictly less than zero, don't want to include zero
		isMiss := !isInsideSpread && !isAbstainVote

		validatorPerformance := validatorPerformances[v.Voter.String()]

		switch {
		case isInsideSpread:
			validatorPerformance.RewardWeight += v.Power
			validatorPerformance.WinCount++
		case isMiss:
			validatorPerformance.MissCount++
		case isAbstainVote:
			validatorPerformance.AbstainCount++
		}

		validatorPerformances[v.Voter.String()] = validatorPerformance
	}

	return weightedMedian
}

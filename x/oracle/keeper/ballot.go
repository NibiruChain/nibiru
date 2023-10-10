package keeper

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

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

	for _, value := range k.Votes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
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
	for _, prevote := range k.Prevotes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
		valAddr, aggregatePrevote := prevote.Key, prevote.Value
		if ctx.BlockHeight() >= int64(aggregatePrevote.SubmitBlock+votePeriod) {
			err := k.Prevotes.Delete(ctx, valAddr)
			if err != nil {
				k.Logger(ctx).Error("failed to delete prevote", "error", err)
			}
		}
	}

	// Clear all aggregate votes
	for _, valAddr := range k.Votes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Keys() {
		err := k.Votes.Delete(ctx, valAddr)
		if err != nil {
			k.Logger(ctx).Error("failed to delete vote", "error", err)
		}
	}
}

// isPassingVoteThreshold ballot is passing the threshold amount of voting power
func isPassingVoteThreshold(
	votes types.ExchangeRateVotes, thresholdVotingPower sdkmath.Int, minVoters uint64,
) bool {
	totalPower := sdk.NewInt(votes.Power())
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

// removeInvalidVotes removes the ballots which have not reached the vote
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
	totalBondedPower := sdk.TokensToConsensusPower(
		k.StakingKeeper.TotalBondedTokens(ctx), k.StakingKeeper.PowerReduction(ctx),
	)

	// Iterate through sorted keys for deterministic ordering.
	orderedBallotsMap := omap.OrderedMap_Pair[types.ExchangeRateVotes](pairVotes)
	for pair := range orderedBallotsMap.Range() {
		// If pair is not whitelisted, or the ballot for it has failed, then skip
		// and remove it from pairBallotsMap for iteration efficiency
		if !whitelistedPairs.Has(pair) {
			delete(pairVotes, pair)
		}

		// If the ballot is not passed, remove it from the whitelistedPairs set
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
	rewardBand sdk.Dec,
	validatorPerformances types.ValidatorPerformances,
) sdk.Dec {
	weightedMedian := votes.WeightedMedianWithAssertion()
	standardDeviation := votes.StandardDeviation(weightedMedian)
	rewardSpread := weightedMedian.Mul(rewardBand.QuoInt64(2))

	if standardDeviation.GT(rewardSpread) {
		rewardSpread = standardDeviation
	}

	for _, v := range votes {
		// Filter ballot winners & abstain voters
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

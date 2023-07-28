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

// groupBallotsByPair takes a collection of votes and organizes them by their
// associated asset pair. This method only considers votes from active validators
// and disregards votes from validators that are not in the provided validator set.
//
// Note that any abstain votes (votes with a non-positive exchange rate) are
// assigned zero vote power. This function then returns a map where each
// asset pair is associated with its collection of ExchangeRateBallots.
func (k Keeper) groupBallotsByPair(
	ctx sdk.Context,
	validatorsPerformance types.ValidatorPerformances,
) (pairBallotsMap map[asset.Pair]types.ExchangeRateBallots) {
	pairBallotsMap = map[asset.Pair]types.ExchangeRateBallots{}

	for _, value := range k.Votes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
		voterAddr, aggregateVote := value.Key, value.Value

		// organize ballot only for the active validators
		if validatorPerformance, exists := validatorsPerformance[aggregateVote.Voter]; exists {
			for _, exchangeRateTuple := range aggregateVote.ExchangeRateTuples {
				power := validatorPerformance.Power
				if !exchangeRateTuple.ExchangeRate.IsPositive() {
					// Make the power of abstain vote zero
					power = 0
				}

				pairBallotsMap[exchangeRateTuple.Pair] = append(pairBallotsMap[exchangeRateTuple.Pair],
					types.NewExchangeRateBallot(
						exchangeRateTuple.ExchangeRate,
						exchangeRateTuple.Pair,
						voterAddr,
						power,
					),
				)
			}
		}
	}

	return
}

// clearVotesAndPreVotes clears all tallied prevotes and votes from the store
func (k Keeper) clearVotesAndPreVotes(ctx sdk.Context, votePeriod uint64) {
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
	ballots types.ExchangeRateBallots, thresholdVotingPower sdkmath.Int, minVoters uint64,
) bool {
	ballotPower := sdk.NewInt(ballots.Power())
	if ballotPower.IsZero() {
		return false
	}

	if ballotPower.LT(thresholdVotingPower) {
		return false
	}

	if ballots.NumValidVoters() < minVoters {
		return false
	}

	return true
}

// removeInvalidBallots removes the ballots which have not reached the vote
// threshold or which are not part of the whitelisted pairs anymore: example
// when params change during a vote period but some votes were already made.
//
// ALERT: This function mutates pairBallotMap slice, it removes the ballot for
// the pair which is not passing the threshold or which is not whitelisted
// anymore.
func (k Keeper) removeInvalidBallots(
	ctx sdk.Context,
	pairBallotsMap map[asset.Pair]types.ExchangeRateBallots,
) (map[asset.Pair]types.ExchangeRateBallots, set.Set[asset.Pair]) {
	whitelistedPairs := set.New(k.GetWhitelistedPairs(ctx)...)

	totalBondedPower := sdk.TokensToConsensusPower(
		k.StakingKeeper.TotalBondedTokens(ctx), k.StakingKeeper.PowerReduction(ctx),
	)
	thresholdVotingPower := k.VoteThreshold(ctx).MulInt64(totalBondedPower).RoundInt()
	minVoters := k.MinVoters(ctx)

	// Iterate through sorted keys for deterministic ordering.
	orderedBallotsMap := omap.OrderedMap_Pair[types.ExchangeRateBallots](pairBallotsMap)
	for pair := range orderedBallotsMap.Range() {
		ballots := pairBallotsMap[pair]
		// If pair is not whitelisted, or the ballot for it has failed, then skip
		// and remove it from pairBallotsMap for iteration efficiency
		if _, exists := whitelistedPairs[pair]; !exists {
			delete(pairBallotsMap, pair)
			continue
		}

		// If the ballot is not passed, remove it from the whitelistedPairs set
		// to prevent slashing validators who did valid vote.
		if !isPassingVoteThreshold(ballots, thresholdVotingPower, minVoters) {
			delete(whitelistedPairs, pair)
			delete(pairBallotsMap, pair)
			continue
		}
	}

	return pairBallotsMap, whitelistedPairs
}

// Tally calculates the median and returns it. Sets the set of voters to be rewarded, i.e. voted within
// a reasonable spread from the weighted median to the store
//
// ALERT: This function mutates validatorPerformances slice based on the votes made by the validators.
func Tally(ballots types.ExchangeRateBallots, rewardBand sdk.Dec, validatorPerformances types.ValidatorPerformances) sdk.Dec {
	weightedMedian := ballots.WeightedMedianWithAssertion()
	standardDeviation := ballots.StandardDeviation(weightedMedian)
	rewardSpread := weightedMedian.Mul(rewardBand.QuoInt64(2))

	if standardDeviation.GT(rewardSpread) {
		rewardSpread = standardDeviation
	}

	for _, ballot := range ballots {
		// Filter ballot winners & abstain voters
		voteInsideSpread := ballot.ExchangeRate.GTE(weightedMedian.Sub(rewardSpread)) &&
			ballot.ExchangeRate.LTE(weightedMedian.Add(rewardSpread))
		isAbstainVote := !ballot.ExchangeRate.IsPositive()

		if voteInsideSpread || isAbstainVote {
			voterAddr := ballot.Voter.String()

			validatorPerformance := validatorPerformances[voterAddr]
			validatorPerformance.RewardWeight += ballot.Power
			validatorPerformance.WinCount++
			validatorPerformances[voterAddr] = validatorPerformance
		}
	}

	return weightedMedian
}

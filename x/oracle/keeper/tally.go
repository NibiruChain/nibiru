package keeper

import (
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// Tally calculates the median and returns it. Sets the set of voters to be rewarded, i.e. voted within
// a reasonable spread from the weighted median to the store
//
// ALERT: This function mutates validatorPerformances slice based on the votes made by the validators.
func Tally(ballots types.ExchangeRateBallots, rewardBand sdk.Dec, validatorPerformances types.ValidatorPerformances) sdk.Dec {
	sort.Sort(ballots)

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

package keeper

import (
	"sort"

	"github.com/NibiruChain/nibiru/collections"

	"github.com/NibiruChain/nibiru/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// OrganizeBallotByPair collects all oracle votes for the period, categorized by the votes' pair parameter
func (k Keeper) OrganizeBallotByPair(ctx sdk.Context, validatorsPerformance map[string]types.ValidatorPerformance) (ballots map[string]types.ExchangeRateBallot) {
	ballots = map[string]types.ExchangeRateBallot{}

	// Organize aggregate votes
	aggregateHandler := func(voterAddr sdk.ValAddress, vote types.AggregateExchangeRateVote) (stop bool) {
		// organize ballot only for the active validators
		if claim, ok := validatorsPerformance[vote.Voter]; ok {
			for _, tuple := range vote.ExchangeRateTuples {
				power := claim.Power
				if !tuple.ExchangeRate.IsPositive() {
					// Make the power of abstain vote zero
					power = 0
				}

				ballots[tuple.Pair] = append(ballots[tuple.Pair],
					types.NewBallotVoteForTally(
						tuple.ExchangeRate,
						tuple.Pair,
						voterAddr,
						power,
					),
				)
			}
		}

		return false
	}

	for _, vote := range k.Votes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
		aggregateHandler(vote.Key, vote.Value)
	}

	// sort created ballot
	for pair, ballot := range ballots {
		sort.Sort(ballot)
		ballots[pair] = ballot
	}

	return
}

// ClearBallots clears all tallied prevotes and votes from the store
func (k Keeper) ClearBallots(ctx sdk.Context, votePeriod uint64) {
	// Clear all aggregate prevotes
	for _, prevote := range k.Prevotes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
		if ctx.BlockHeight() > int64(prevote.Value.SubmitBlock+votePeriod) {
			err := k.Prevotes.Delete(ctx, prevote.Key)
			if err != nil {
				panic(err)
			}
		}
	}

	// Clear all aggregate votes
	for _, voteKey := range k.Votes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Keys() {
		err := k.Votes.Delete(ctx, voteKey)
		if err != nil {
			panic(err)
		}
	}
}

// ApplyWhitelist updates the whitelist by detecting possible changes between
// the current vote targets and the current updated whitelist.
func (k Keeper) ApplyWhitelist(ctx sdk.Context, whitelist types.PairList, voteTargets map[string]struct{}) {
	// check is there any update in whitelist params
	updateRequired := false
	// fast path
	if len(voteTargets) != len(whitelist) {
		updateRequired = true
		// slow path, we need to check differences
	} else {
		for _, pair := range whitelist {
			_, exists := voteTargets[pair.Name]
			if !exists {
				updateRequired = true
				break
			}
		}
	}

	if updateRequired {
		for _, p := range k.Pairs.Iterate(ctx, collections.Range[string]{}).Keys() {
			k.Pairs.Delete(ctx, p)
		}
		for _, pair := range whitelist {
			k.Pairs.Insert(ctx, pair.Name)
		}
	}
}

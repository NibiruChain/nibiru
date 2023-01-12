package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// groupBallotsByPair groups votes by pair and removes votes that are not part of
// the validator set.
//
// NOTE: **Make abstain votes to have zero vote power**
func (k Keeper) groupBallotsByPair(
	ctx sdk.Context,
	validatorsPerformance map[string]types.ValidatorPerformance,
) (pairBallotsMap map[string]types.ExchangeRateBallots) {
	pairBallotsMap = map[string]types.ExchangeRateBallots{}

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

// updateWhitelist updates the whitelist by detecting possible changes between
// the current vote targets and the current updated whitelist.
func (k Keeper) updateWhitelist(ctx sdk.Context, paramsWhitelist []string, currentWhitelist map[string]struct{}) {
	updateRequired := false

	if len(currentWhitelist) != len(paramsWhitelist) {
		updateRequired = true
	} else {
		for _, pair := range paramsWhitelist {
			_, exists := currentWhitelist[pair]
			if !exists {
				updateRequired = true
				break
			}
		}
	}

	if updateRequired {
		for _, p := range k.WhitelistedPairs.Iterate(ctx, collections.Range[string]{}).Keys() {
			k.WhitelistedPairs.Delete(ctx, p)
		}
		for _, pair := range paramsWhitelist {
			k.WhitelistedPairs.Insert(ctx, pair)
		}
	}
}

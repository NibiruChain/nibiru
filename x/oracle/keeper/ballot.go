package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// groupBallotsByPair groups votes by pair and removes votes that are not part of
// the validator set.
//
// NOTE: **Make abstain votes to have zero vote power**
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
func (k Keeper) updateWhitelist(ctx sdk.Context, nextWhitelist []asset.Pair, currentWhitelist map[asset.Pair]struct{}) {
	updateRequired := false

	if len(currentWhitelist) != len(nextWhitelist) {
		updateRequired = true
	} else {
		for _, pair := range nextWhitelist {
			_, exists := currentWhitelist[pair]
			if !exists {
				updateRequired = true
				break
			}
		}
	}

	if updateRequired {
		for _, p := range k.WhitelistedPairs.Iterate(ctx, collections.Range[asset.Pair]{}).Keys() {
			k.WhitelistedPairs.Delete(ctx, p)
		}
		for _, pair := range nextWhitelist {
			k.WhitelistedPairs.Insert(ctx, pair)
		}
	}
}

// RemoveInvalidBallots removes the ballots which have not reached the vote threshold
// or which are not part of the whitelisted pairs anymore: example when params change during a vote period
// but some votes were already made.
//
// ALERT: This function mutates pairBallotMap slice, it removes the ballot for the pair which is not passing the threshold
// or which is not whitelisted anymore.
func (k Keeper) RemoveInvalidBallots(
	ctx sdk.Context,
	pairBallotsMap map[asset.Pair]types.ExchangeRateBallots,
) (map[asset.Pair]types.ExchangeRateBallots, map[asset.Pair]struct{}) {
	whitelistedPairs := k.GetWhitelistedPairs(ctx)

	whitelistedPairsMap := make(map[asset.Pair]struct{}, len(whitelistedPairs))
	for _, pair := range whitelistedPairs {
		whitelistedPairsMap[pair] = struct{}{}
	}

	totalBondedPower := sdk.TokensToConsensusPower(k.StakingKeeper.TotalBondedTokens(ctx), k.StakingKeeper.PowerReduction(ctx))
	thresholdPower := k.VoteThreshold(ctx).MulInt64(totalBondedPower).RoundInt()

	for pair, ballots := range pairBallotsMap {
		// Ignore not whitelisted pairs
		if _, exists := whitelistedPairsMap[pair]; !exists {
			delete(pairBallotsMap, pair)
			continue
		}

		// If the ballot is not passed, remove it from the whitelistedPairs set
		// to prevent slashing validators who did valid vote.
		if !isPassingVoteThreshold(ballots, thresholdPower) {
			delete(whitelistedPairsMap, pair)
			delete(pairBallotsMap, pair)
			continue
		}
	}

	return pairBallotsMap, whitelistedPairsMap
}

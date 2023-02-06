package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// groupBallotsByPair groups votes by pair and removes votes that are not part of
// the validator set.
//
// NOTE: **Make abstain votes to have zero vote power**
func (k Keeper) groupBallotsByPair(
	ctx sdk.Context,
	validatorsPerformance types.ValidatorPerformances,
) (voteMap types.VoteMap) {
	voteMap = map[asset.Pair]types.ExchangeRateBallots{}

	for _, value := range k.Votes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
		valAddr, aggregateVote := value.Key, value.Value

		// organize ballot only for the active validators
		if validatorPerformance, exists := validatorsPerformance[aggregateVote.Voter]; exists {
			for _, exchangeRateTuple := range aggregateVote.ExchangeRateTuples {
				power := validatorPerformance.Power
				if !exchangeRateTuple.ExchangeRate.IsPositive() {
					// Make the power of abstain vote zero
					power = 0
				}

				voteMap[exchangeRateTuple.Pair] = append(voteMap[exchangeRateTuple.Pair],
					types.NewExchangeRateBallot(
						exchangeRateTuple.ExchangeRate,
						exchangeRateTuple.Pair,
						valAddr,
						power,
					),
				)
			}
		}
	}

	return
}

// clearVotesAndPreVotes clears all prevotes and votes from the store
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

// isPassingVoteThreshold ballot is passing the threshold amount of voting power
func isPassingVoteThreshold(ballots types.ExchangeRateBallots, thresholdPower sdk.Int) bool {
	ballotPower := sdk.NewInt(ballots.Power())
	return !ballotPower.IsZero() && ballotPower.GTE(thresholdPower)
}

// removeInvalidBallots removes the ballots which have not reached the vote threshold
// or which are not part of the whitelisted pairs anymore: example when params change during a vote period
// but some votes were already made.
//
// ALERT: This function mutates pairBallotMap slice, it removes the ballot for the pair which is not passing the threshold
// or which is not whitelisted anymore.
func (k Keeper) removeInvalidBallots(
	ctx sdk.Context,
	voteMap types.VoteMap,
) (types.VoteMap, set.Set[asset.Pair]) {
	whitelistedPairs := set.New(k.GetWhitelistedPairs(ctx)...)
	totalBondedPower := sdk.TokensToConsensusPower(k.StakingKeeper.TotalBondedTokens(ctx), k.StakingKeeper.PowerReduction(ctx))
	thresholdPower := k.VoteThreshold(ctx).MulInt64(totalBondedPower).RoundInt()

	for pair, ballots := range voteMap {
		// Ignore not whitelisted pairs
		if !whitelistedPairs.Has(pair) {
			delete(voteMap, pair)
			continue
		}

		// If the ballot is not passed, remove it from the whitelistedPairs set
		// to prevent slashing validators who did valid vote.
		if !isPassingVoteThreshold(ballots, thresholdPower) {
			delete(whitelistedPairs, pair)
			delete(voteMap, pair)
			continue
		}
	}

	return voteMap, whitelistedPairs
}

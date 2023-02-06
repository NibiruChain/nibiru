package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// UpdateExchangeRates updates the ExchangeRates, this is supposed to be executed on EndBlock.
func (k Keeper) UpdateExchangeRates(ctx sdk.Context) {
	k.Logger(ctx).Info("processing validator price votes")
	k.resetExchangeRates(ctx)

	validatorPerformances := k.newValidatorPerformances(ctx)
	voteMap, whitelistedPairs := k.getVoteMapAndWhitelistedPairs(ctx, validatorPerformances)

	k.countVotesAndUpdateExchangeRates(ctx, voteMap, validatorPerformances)
	k.registerMissedVotes(ctx, whitelistedPairs, validatorPerformances)
	k.rewardBallotWinners(ctx, whitelistedPairs, validatorPerformances)

	params := k.GetParams(ctx)
	k.clearVotesAndPreVotes(ctx, params.VotePeriod)
	k.updateWhitelist(ctx, params.Whitelist, whitelistedPairs)
}

// registerMissedVotes it parses all validators performance and increases the missed vote of those that did not vote.
func (k Keeper) registerMissedVotes(ctx sdk.Context, whitelistedPairs map[asset.Pair]struct{}, validatorPerformances types.ValidatorPerformances) {
	for _, validatorPerformance := range validatorPerformances {
		if int(validatorPerformance.WinCount) == len(whitelistedPairs) {
			continue
		}

		k.MissCounters.Insert(ctx, validatorPerformance.ValAddress, k.MissCounters.GetOr(ctx, validatorPerformance.ValAddress, 0)+1)
		k.Logger(ctx).Info("vote miss", "validator", validatorPerformance.ValAddress.String())
	}
}

// countVotesAndUpdateExchangeRates processes the votes and updates the ExchangeRates based on the results.
func (k Keeper) countVotesAndUpdateExchangeRates(
	ctx sdk.Context,
	voteMap map[asset.Pair]types.ExchangeRateBallots,
	validatorPerformances types.ValidatorPerformances,
) {
	params := k.GetParams(ctx)

	for pair, ballots := range voteMap {
		exchangeRate := Tally(ballots, params.RewardBand, validatorPerformances)

		k.SetPrice(ctx, pair, exchangeRate)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(types.EventTypeExchangeRateUpdate,
				sdk.NewAttribute(types.AttributeKeyPair, pair.String()),
				sdk.NewAttribute(types.AttributeKeyExchangeRate, exchangeRate.String()),
			),
		)
	}
}

// getVoteMapAndWhitelistedPairs returns a map of pairs and ballots excluding invalid Ballots
// and a set of all whitelisted pairs.
func (k Keeper) getVoteMapAndWhitelistedPairs(
	ctx sdk.Context,
	validatorPerformances types.ValidatorPerformances,
) (voteMap types.VoteMap, whitelistedPairs set.Set[asset.Pair]) {
	voteMap = k.groupBallotsByPair(ctx, validatorPerformances)

	return k.removeInvalidBallots(ctx, voteMap)
}

// resetExchangeRates removes all exchange rates from the state
func (k Keeper) resetExchangeRates(ctx sdk.Context) {
	for _, key := range k.ExchangeRates.Iterate(ctx, collections.Range[asset.Pair]{}).Keys() {
		err := k.ExchangeRates.Delete(ctx, key)
		if err != nil {
			panic(err)
		}
	}
}

// newValidatorPerformances creates a new map of validators and their performance, excluding validators that are
// not bonded.
func (k Keeper) newValidatorPerformances(ctx sdk.Context) types.ValidatorPerformances {
	validatorPerformances := make(types.ValidatorPerformances)

	maxValidators := k.StakingKeeper.MaxValidators(ctx)
	powerReduction := k.StakingKeeper.PowerReduction(ctx)

	iterator := k.StakingKeeper.ValidatorsPowerStoreIterator(ctx)
	defer iterator.Close()

	for i := 0; iterator.Valid() && i < int(maxValidators); iterator.Next() {
		validator := k.StakingKeeper.Validator(ctx, iterator.Value())

		// exclude not bonded
		if !validator.IsBonded() {
			continue
		}

		valAddr := validator.GetOperator()
		validatorPerformances[valAddr.String()] = types.ValidatorPerformance{
			Power:        validator.GetConsensusPower(powerReduction),
			ValAddress:   valAddr,
			RewardWeight: 0,
			WinCount:     0,
		}

		i++
	}

	return validatorPerformances
}

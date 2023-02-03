package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// UpdateExchangeRates updates the ExchangeRates, this is supposed to be executed on EndBlock.
func (k Keeper) UpdateExchangeRates(ctx sdk.Context) {
	k.Logger(ctx).Info("processing validator price votes")
	k.resetExchangeRates(ctx)

	validatorPerformanceMap := k.newValidatorPerformanceMap(ctx)
	pairBallotsMap, whitelistedPairs := k.getPairBallotsMapAndWhitelistedPairs(ctx, validatorPerformanceMap)

	k.countVotesAndUpdateExchangeRates(ctx, pairBallotsMap, validatorPerformanceMap)
	k.registerMissedVotes(ctx, whitelistedPairs, validatorPerformanceMap)
	k.rewardBallotWinners(ctx, whitelistedPairs, validatorPerformanceMap)

	params := k.GetParams(ctx)
	k.clearVotesAndPreVotes(ctx, params.VotePeriod)
	k.updateWhitelist(ctx, params.Whitelist, whitelistedPairs)
}

// registerMissedVotes it parses all validators performance and increases the missed vote of those that did not vote.
func (k Keeper) registerMissedVotes(ctx sdk.Context, whitelistedPairs map[asset.Pair]struct{}, validatorPerformanceMap map[string]types.ValidatorPerformance) {
	for _, validatorPerformance := range validatorPerformanceMap {
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
	pairBallotsMap map[asset.Pair]types.ExchangeRateBallots,
	validatorPerformanceMap map[string]types.ValidatorPerformance,
) {
	params := k.GetParams(ctx)

	for pair, ballots := range pairBallotsMap {
		exchangeRate := Tally(ballots, params.RewardBand, validatorPerformanceMap)

		k.SetPrice(ctx, pair, exchangeRate)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(types.EventTypeExchangeRateUpdate,
				sdk.NewAttribute(types.AttributeKeyPair, pair.String()),
				sdk.NewAttribute(types.AttributeKeyExchangeRate, exchangeRate.String()),
			),
		)
	}
}

// getPairBallotsMapAndWhitelistedPairs returns a map of pairs and ballots excluding invalid Ballots
// and a map with all whitelisted pairs.
func (k Keeper) getPairBallotsMapAndWhitelistedPairs(
	ctx sdk.Context,
	validatorPerformanceMap map[string]types.ValidatorPerformance,
) (pairBallotsMap map[asset.Pair]types.ExchangeRateBallots, whitelistedPairsMap map[asset.Pair]struct{}) {
	pairBallotsMap = k.groupBallotsByPair(ctx, validatorPerformanceMap)

	return k.RemoveInvalidBallots(ctx, pairBallotsMap)
}

// getWhitelistedPairs returns a map containing all the pairs as the key.
func (k Keeper) getWhitelistedPairs(ctx sdk.Context) map[asset.Pair]struct{} {
	whitelistedPairs := make(map[asset.Pair]struct{})
	for _, p := range k.GetWhitelistedPairs(ctx) {
		whitelistedPairs[p] = struct{}{}
	}

	return whitelistedPairs
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

// newValidatorPerformanceMap creates a new map of validators and their performance, excluding validators that are
// not bonded.
func (k Keeper) newValidatorPerformanceMap(ctx sdk.Context) map[string]types.ValidatorPerformance {
	validatorPerformanceMap := make(map[string]types.ValidatorPerformance)

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
		validatorPerformanceMap[valAddr.String()] = types.NewValidatorPerformance(
			validator.GetConsensusPower(powerReduction), valAddr,
		)
		i++
	}

	return validatorPerformanceMap
}

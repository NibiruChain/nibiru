package keeper

import (
	"github.com/NibiruChain/nibiru/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// UpdateExchangeRates updates the ExchangeRates, this is supposed to be executed on EndBlock.
func (k Keeper) UpdateExchangeRates(ctx sdk.Context) {
	k.Logger(ctx).Info("processing validator price votes")
	k.resetExchangeRates(ctx)

	validatorPerformanceMap := k.buildEmptyValidatorPerformanceMap(ctx)
	pairBallotMap, pairsMap := k.getPairBallotMapAndPairsMap(ctx, validatorPerformanceMap)

	k.countVotesAndUpdateExchangeRates(ctx, pairBallotMap, validatorPerformanceMap)

	//---------------------------
	// Do miss counting & slashing
	params := k.GetParams(ctx)
	voteTargetsLen := len(pairsMap)
	for _, claim := range validatorPerformanceMap {
		// Skip abstain & valid voters
		if int(claim.WinCount) == voteTargetsLen {
			continue
		}

		// Increase miss counter
		k.MissCounters.Insert(ctx, claim.ValAddress, k.MissCounters.GetOr(ctx, claim.ValAddress, 0)+1)
		k.Logger(ctx).Info("vote miss", "validator", claim.ValAddress.String())
	}

	k.rewardBallotWinners(ctx, pairsMap, validatorPerformanceMap)
	k.clearBallots(ctx, params.VotePeriod)
	k.applyWhitelist(ctx, params.Whitelist, pairsMap)
}

// countVotesAndUpdateExchangeRates processes the votes and updates the ExchangeRates based on the results.
func (k Keeper) countVotesAndUpdateExchangeRates(
	ctx sdk.Context,
	pairBallotMap map[string]types.ExchangeRateBallot,
	validatorPerformanceMap map[string]types.ValidatorPerformance,
) {
	params := k.GetParams(ctx)

	for pair, ballot := range pairBallotMap {
		// Get weighted median of cross exchange rates
		exchangeRate := Tally(ballot, params.RewardBand, validatorPerformanceMap)

		// Set the exchange rate, emit ABCI event
		k.ExchangeRates.Insert(ctx, pair, exchangeRate)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(types.EventTypeExchangeRateUpdate,
				sdk.NewAttribute(types.AttributeKeyPair, pair),
				sdk.NewAttribute(types.AttributeKeyExchangeRate, exchangeRate.String()),
			),
		)
	}
}

// getPairBallotMapAndPairsMap returns a map of pairs and ballots excluding invalid Ballots and a map with all the pairs.
func (k Keeper) getPairBallotMapAndPairsMap(
	ctx sdk.Context,
	validatorPerformanceMap map[string]types.ValidatorPerformance,
) (map[string]types.ExchangeRateBallot, map[string]struct{}) {
	pairBallotMap := k.mapBallotByPair(ctx, validatorPerformanceMap)

	updatedPairBallotMap, pairsMap := k.RemoveInvalidBallots(ctx, pairBallotMap)

	return updatedPairBallotMap, pairsMap
}

// getPairsMap returns a map containing all the pairs as the key.
func (k Keeper) getPairsMap(ctx sdk.Context) map[string]struct{} {
	pairsMap := make(map[string]struct{})
	for _, p := range k.Pairs.Iterate(ctx, collections.Range[string]{}).Keys() {
		pairsMap[p] = struct{}{}
	}

	return pairsMap
}

// resetExchangeRates removes all exchange rates from the state
func (k Keeper) resetExchangeRates(ctx sdk.Context) {
	for _, key := range k.ExchangeRates.Iterate(ctx, collections.Range[string]{}).Keys() {
		err := k.ExchangeRates.Delete(ctx, key)
		if err != nil {
			panic(err)
		}
	}
}

// buildEmptyValidatorPerformanceMap creates a new map of validators and their performance, excluding validators that are
// not bonded.
func (k Keeper) buildEmptyValidatorPerformanceMap(ctx sdk.Context) map[string]types.ValidatorPerformance {
	validatorPerformanceMap := make(map[string]types.ValidatorPerformance)

	maxValidators := k.StakingKeeper.MaxValidators(ctx)
	powerReduction := k.StakingKeeper.PowerReduction(ctx)

	iterator := k.StakingKeeper.ValidatorsPowerStoreIterator(ctx)
	defer iterator.Close()

	i := 0
	for ; iterator.Valid() && i < int(maxValidators); iterator.Next() {
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

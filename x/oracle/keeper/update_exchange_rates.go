package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// UpdateExchangeRates updates the ExchangeRates, this is supposed to be executed on EndBlock.
func (k Keeper) UpdateExchangeRates(ctx sdk.Context) {
	k.Logger(ctx).Info("processing validator price votes")
	k.resetExchangeRates(ctx)

	validatorPerformanceMap := k.newValidatorPerformanceMap(ctx)
	pairBallotMap, whitelistedPairsMap := k.getPairBallotMapAndWhitelistedPairsMap(ctx, validatorPerformanceMap)

	k.countVotesAndUpdateExchangeRates(ctx, pairBallotMap, validatorPerformanceMap)
	k.registerMissedVotes(ctx, whitelistedPairsMap, validatorPerformanceMap)

	k.rewardBallotWinners(ctx, whitelistedPairsMap, validatorPerformanceMap)

	params := k.GetParams(ctx)
	k.clearBallots(ctx, params.VotePeriod)
	k.applyWhitelist(ctx, params.Whitelist, whitelistedPairsMap)
}

// registerMissedVotes it parses all validators performance and increases the missed vote of those that did not vote.
func (k Keeper) registerMissedVotes(ctx sdk.Context, whitelistedPairsMap map[string]struct{}, validatorPerformanceMap map[string]types.ValidatorPerformance) {
	whitelistedPairsLen := len(whitelistedPairsMap)
	for _, validatorPerformance := range validatorPerformanceMap {
		if int(validatorPerformance.WinCount) == whitelistedPairsLen {
			continue
		}

		k.MissCounters.Insert(ctx, validatorPerformance.ValAddress, k.MissCounters.GetOr(ctx, validatorPerformance.ValAddress, 0)+1)
		k.Logger(ctx).Info("vote miss", "validator", validatorPerformance.ValAddress.String())
	}
}

// countVotesAndUpdateExchangeRates processes the votes and updates the ExchangeRates based on the results.
func (k Keeper) countVotesAndUpdateExchangeRates(
	ctx sdk.Context,
	pairBallotMap map[string]types.ExchangeRateBallot,
	validatorPerformanceMap map[string]types.ValidatorPerformance,
) {
	params := k.GetParams(ctx)

	for pair, ballot := range pairBallotMap {
		exchangeRate := Tally(ballot, params.RewardBand, validatorPerformanceMap)

		k.ExchangeRates.Insert(ctx, pair, exchangeRate)
		k.PriceSnapshots.Insert(ctx, collections.Join(pair, ctx.BlockTime()), types.PriceSnapshot{
			Pair:        pair,
			Price:       exchangeRate,
			TimestampMs: ctx.BlockTime().UnixMilli(),
		})

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(types.EventTypeExchangeRateUpdate,
				sdk.NewAttribute(types.AttributeKeyPair, pair),
				sdk.NewAttribute(types.AttributeKeyExchangeRate, exchangeRate.String()),
			),
		)
	}
}

// getPairBallotMapAndWhitelistedPairsMap returns a map of pairs and ballots excluding invalid Ballots
// and a map with all whitelisted pairs.
func (k Keeper) getPairBallotMapAndWhitelistedPairsMap(
	ctx sdk.Context,
	validatorPerformanceMap map[string]types.ValidatorPerformance,
) (map[string]types.ExchangeRateBallot, map[string]struct{}) {
	pairBallotMap := k.mapBallotByPair(ctx, validatorPerformanceMap)

	return k.RemoveInvalidBallots(ctx, pairBallotMap)
}

// getWhitelistedPairsMap returns a map containing all the pairs as the key.
func (k Keeper) getWhitelistedPairsMap(ctx sdk.Context) map[string]struct{} {
	pairsMap := make(map[string]struct{})
	for _, p := range k.GetWhitelistedPairs(ctx) {
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

// newValidatorPerformanceMap creates a new map of validators and their performance, excluding validators that are
// not bonded.
func (k Keeper) newValidatorPerformanceMap(ctx sdk.Context) map[string]types.ValidatorPerformance {
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

package keeper

import (
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// UpdateExchangeRates updates the ExchangeRates, this is supposed to be executed on EndBlock.
func (k Keeper) UpdateExchangeRates(ctx sdk.Context) {
	k.Logger(ctx).Info("processing validator price votes")

	validatorPerformances := k.newValidatorPerformances(ctx)
	pairBallotsMap, whitelistedPairs := k.getPairBallotsMapAndWhitelistedPairs(ctx, validatorPerformances)

	k.resetExchangeRates(ctx, pairBallotsMap)
	k.countVotesAndUpdateExchangeRates(ctx, pairBallotsMap, validatorPerformances)

	k.registerMissedVotes(ctx, whitelistedPairs, validatorPerformances)
	k.rewardBallotWinners(ctx, validatorPerformances)

	params, _ := k.Params.Get(ctx)
	k.clearVotesAndPreVotes(ctx, params.VotePeriod)
	k.updateWhitelist(ctx, params.Whitelist, whitelistedPairs)
}

// registerMissedVotes it parses all validators performance and increases the missed vote of those that did not vote.
func (k Keeper) registerMissedVotes(ctx sdk.Context, whitelistedPairs set.Set[asset.Pair], validatorPerformances types.ValidatorPerformances) {
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
	pairBallotsMap map[asset.Pair]types.ExchangeRateBallots,
	validatorPerformances types.ValidatorPerformances,
) {
	rewardBand := k.RewardBand(ctx)

	// Iterate through sorted keys for deterministic ordering.
	// For more info, see: https://github.com/NibiruChain/nibiru/issues/1374#issue-1715353299
	var pairs []string
	for pair := range pairBallotsMap {
		pairs = append(pairs, pair.String())
	}
	sort.Strings(pairs)
	for _, pairStr := range pairs {
		pair := asset.Pair(pairStr)
		ballots := pairBallotsMap[pair]
		exchangeRate := Tally(ballots, rewardBand, validatorPerformances)

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
	validatorPerformances types.ValidatorPerformances,
) (pairBallotsMap map[asset.Pair]types.ExchangeRateBallots, whitelistedPairsMap set.Set[asset.Pair]) {
	pairBallotsMap = k.groupBallotsByPair(ctx, validatorPerformances)

	return k.removeInvalidBallots(ctx, pairBallotsMap)
}

// resetExchangeRates removes all exchange rates from the state
// We remove the price for pair with expired prices or valid ballots
func (k Keeper) resetExchangeRates(ctx sdk.Context, pairBallotsMap map[asset.Pair]types.ExchangeRateBallots) {
	params, _ := k.Params.Get(ctx)
	expirationBlocks := params.ExpirationBlocks

	for _, key := range k.ExchangeRates.Iterate(ctx, collections.Range[asset.Pair]{}).Keys() {
		_, validBallot := pairBallotsMap[key]
		exchangeRate, _ := k.ExchangeRates.Get(ctx, key)
		isExpired := exchangeRate.CreatedBlock+expirationBlocks <= uint64(ctx.BlockHeight())

		if validBallot || isExpired {
			err := k.ExchangeRates.Delete(ctx, key)
			if err != nil {
				k.Logger(ctx).Error("failed to delete exchange rate", "pair", key.String(), "error", err)
			}
		}
	}
}

// newValidatorPerformances creates a new map of validators and their performance, excluding validators that are
// not bonded.
func (k Keeper) newValidatorPerformances(ctx sdk.Context) types.ValidatorPerformances {
	validatorPerformances := make(map[string]types.ValidatorPerformance)

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
		validatorPerformances[valAddr.String()] = types.NewValidatorPerformance(
			validator.GetConsensusPower(powerReduction), valAddr,
		)
		i++
	}

	return validatorPerformances
}

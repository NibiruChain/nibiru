package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/omap"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// UpdateExchangeRates updates the ExchangeRates, this is supposed to be executed on EndBlock.
func (k Keeper) UpdateExchangeRates(ctx sdk.Context) types.ValidatorPerformances {
	k.Logger(ctx).Info("processing validator price votes")
	validatorPerformances := k.newValidatorPerformances(ctx)
	whitelistedPairs := set.New[asset.Pair](k.GetWhitelistedPairs(ctx)...)

	pairVotes := k.getPairVotes(ctx, validatorPerformances, whitelistedPairs)

	k.clearExchangeRates(ctx, pairVotes)
	k.tallyVotesAndUpdatePrices(ctx, pairVotes, validatorPerformances)

	k.incrementMissCounters(ctx, whitelistedPairs, validatorPerformances)
	k.incrementAbstainsByOmission(ctx, len(whitelistedPairs), validatorPerformances)

	k.rewardWinners(ctx, validatorPerformances)

	params, _ := k.Params.Get(ctx)
	k.clearVotesAndPrevotes(ctx, params.VotePeriod)
	k.refreshWhitelist(ctx, params.Whitelist, whitelistedPairs)

	for _, validatorPerformance := range validatorPerformances {
		_ = ctx.EventManager().EmitTypedEvent(&types.EventValidatorPerformance{
			Validator:    validatorPerformance.ValAddress.String(),
			VotingPower:  validatorPerformance.Power,
			RewardWeight: validatorPerformance.RewardWeight,
			WinCount:     validatorPerformance.WinCount,
			AbstainCount: validatorPerformance.AbstainCount,
			MissCount:    validatorPerformance.MissCount,
		})
	}

	return validatorPerformances
}

// incrementMissCounters it parses all validators performance and increases the
// missed vote of those that did not vote.
func (k Keeper) incrementMissCounters(
	ctx sdk.Context,
	whitelistedPairs set.Set[asset.Pair],
	validatorPerformances types.ValidatorPerformances,
) {
	for _, validatorPerformance := range validatorPerformances {
		if int(validatorPerformance.MissCount) > 0 {
			missCounters, err := k.MissCounters.Get(ctx, validatorPerformance.ValAddress)
			if err != nil {
				missCounters = uint64(0)
			}
			k.MissCounters.Set(
				ctx, validatorPerformance.ValAddress,
				missCounters+uint64(validatorPerformance.MissCount),
			)

			k.Logger(ctx).Info("vote miss", "validator", validatorPerformance.ValAddress.String())
		}
	}
}

func (k Keeper) incrementAbstainsByOmission(
	ctx sdk.Context,
	numPairs int,
	validatorPerformances types.ValidatorPerformances,
) {
	for valAddr, performance := range validatorPerformances {
		omitCount := int64(numPairs) - (performance.WinCount + performance.AbstainCount + performance.MissCount)
		if omitCount > 0 {
			performance.AbstainCount += omitCount
			validatorPerformances[valAddr] = performance
		}
	}
}

// tallyVotesAndUpdatePrices processes the votes and updates the ExchangeRates based on the results.
func (k Keeper) tallyVotesAndUpdatePrices(
	ctx sdk.Context,
	pairVotes map[asset.Pair]types.ExchangeRateVotes,
	validatorPerformances types.ValidatorPerformances,
) {
	rewardBand := k.RewardBand(ctx)
	// Iterate through sorted keys for deterministic ordering.
	orderedPairVotes := omap.OrderedMap_Pair[types.ExchangeRateVotes](pairVotes)
	for pair := range orderedPairVotes.Range() {
		exchangeRate := Tally(pairVotes[pair], rewardBand, validatorPerformances)
		k.SetPrice(ctx, pair, exchangeRate)
	}
}

// getPairVotes returns a map of pairs and votes excluding abstained votes and votes that don't meet the threshold criteria
func (k Keeper) getPairVotes(
	ctx sdk.Context,
	validatorPerformances types.ValidatorPerformances,
	whitelistedPairs set.Set[asset.Pair],
) (pairVotes map[asset.Pair]types.ExchangeRateVotes) {
	pairVotes = k.groupVotesByPair(ctx, validatorPerformances)

	k.removeInvalidVotes(ctx, pairVotes, whitelistedPairs)

	return pairVotes
}

// clearExchangeRates removes all exchange rates from the state
// We remove the price for pair with expired prices or valid votes
func (k Keeper) clearExchangeRates(ctx sdk.Context, pairVotes map[asset.Pair]types.ExchangeRateVotes) {
	params, _ := k.Params.Get(ctx)

	iter, err := k.ExchangeRates.Iterate(ctx, &collections.Range[asset.Pair]{})
	defer iter.Close()

	if err != nil {
		k.Logger(ctx).Error("failed to iterate exchange rates", "error", err)
		return
	}
	keys, err := iter.Keys()
	if err != nil {
		k.Logger(ctx).Error("failed to get exchange rate keys", "error", err)
		return
	}
	for _, key := range keys {
		_, isValid := pairVotes[key]
		previousExchangeRate, _ := k.ExchangeRates.Get(ctx, key)
		isExpired := previousExchangeRate.CreatedBlock+params.ExpirationBlocks <= uint64(ctx.BlockHeight())

		if isValid || isExpired {
			err := k.ExchangeRates.Remove(ctx, key)
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

	maxValidators, err := k.StakingKeeper.MaxValidators(ctx)
	if err != nil {
		k.Logger(ctx).Error("failed getting max validators", "error", err)
		return validatorPerformances
	}

	powerReduction := k.StakingKeeper.PowerReduction(ctx)

	iterator, err := k.StakingKeeper.ValidatorsPowerStoreIterator(ctx)
	if err != nil {
		k.Logger(ctx).Error("failed getting validators power store iterator", "error", err)
		return validatorPerformances
	}

	defer iterator.Close()

	for i := 0; iterator.Valid() && i < int(maxValidators); iterator.Next() {
		validator, err := k.StakingKeeper.Validator(ctx, iterator.Value())
		if err != nil {
			k.Logger(ctx).Error("failed getting validator", "error", err)
			return validatorPerformances
		}

		// exclude not bonded
		if !validator.IsBonded() {
			continue
		}

		valAddr := validator.GetOperator()
		validatorPerformances[valAddr] = types.NewValidatorPerformance(
			validator.GetConsensusPower(powerReduction), sdk.ValAddress(valAddr),
		)
		i++
	}

	return validatorPerformances
}

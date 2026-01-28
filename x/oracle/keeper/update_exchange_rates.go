package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/nutil/asset"
	"github.com/NibiruChain/nibiru/v2/x/nutil/omap"
	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

// UpdateExchangeRates updates the ExchangeRates, this is supposed to be executed on EndBlock.
func (k Keeper) UpdateExchangeRates(ctx sdk.Context) types.ValidatorPerformances {
	k.Logger(ctx).Info("processing validator price votes")
	validatorPerformances := k.newValidatorPerformances(ctx)
	whitelistedPairs := set.New[asset.Pair](k.GetWhitelistedPairs(ctx)...)

	pairVotes := k.getPairVotes(ctx, validatorPerformances, whitelistedPairs)

	k.tallyVotesAndUpdatePrices(ctx, pairVotes, validatorPerformances)

	k.incrementMissCounters(ctx, whitelistedPairs, validatorPerformances)
	k.incrementAbstainsByOmission(ctx, len(whitelistedPairs), validatorPerformances)

	k.rewardWinners(ctx, validatorPerformances)

	params, _ := k.ModuleParams.Get(ctx)
	k.clearVotesAndPrevotes(ctx, params.VotePeriod)
	k.refreshWhitelist(ctx, params.Whitelist, whitelistedPairs)

	// Sort validator addresses for deterministic event emission order.
	// Go map iteration is non-deterministic, which can cause consensus
	// failures if events are emitted in different order on different nodes.
	sortedAddrs := validatorPerformances.SortedAddrs()
	for _, valAddr := range sortedAddrs {
		validatorPerformance := validatorPerformances[valAddr]
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
	// Sort validator addresses for deterministic iteration order.
	// Go map iteration is non-deterministic, which can cause consensus
	// failures if state is written in different order on different nodes.
	sortedAddrs := validatorPerformances.SortedAddrs()

	for _, valAddr := range sortedAddrs {
		validatorPerformance := validatorPerformances[valAddr]
		if int(validatorPerformance.MissCount) > 0 {
			k.MissCounters.Insert(
				ctx, validatorPerformance.ValAddress,
				k.MissCounters.GetOr(ctx, validatorPerformance.ValAddress, 0)+uint64(validatorPerformance.MissCount),
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
	orderedPairVotes := omap.SortedMap_Pair[types.ExchangeRateVotes](pairVotes)
	for pair := range orderedPairVotes.Range() {
		exchangeRate := Tally(pairVotes[pair], rewardBand, validatorPerformances)
		k.SetPrice(ctx, pair, exchangeRate)
	}
}

// getPairVotes returns a map of pairs and votes excluding abstained votes and
// votes that don't meet the threshold criteria
//
// Returns:
//   - pairVotes: A filtered collection of valid votes that have passed all
//     validation criteria. If an asset pair has a value in this map, it is a viable
//     next value for the module's current exchange rate.
func (k Keeper) getPairVotes(
	ctx sdk.Context,
	validatorPerformances types.ValidatorPerformances,
	whitelistedPairs set.Set[asset.Pair],
) (pairVotes map[asset.Pair]types.ExchangeRateVotes) {
	pairVotes = k.groupVotesByPair(ctx, validatorPerformances)

	k.removeInvalidVotes(ctx, pairVotes, whitelistedPairs)

	return pairVotes
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

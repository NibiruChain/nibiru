package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/omap"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// UpdateExchangeRates updates the ExchangeRates, this is supposed to be executed on EndBlock.
func (k Keeper) UpdateExchangeRates(ctx sdk.Context) types.ValidatorPerformances {
	k.Logger(ctx).Info("processing validator price votes")

	validatorPerformances := k.newValidatorPerformances(ctx)
	pairVotes, whitelistedPairs := k.getPairVotesAndWhitelistedPairs(ctx, validatorPerformances)

	k.resetExchangeRates(ctx, pairVotes)
	k.countVotesAndUpdateExchangeRates(ctx, pairVotes, validatorPerformances)

	k.registerMissedVotes(ctx, whitelistedPairs, validatorPerformances)
	k.rewardWinners(ctx, validatorPerformances)

	params, _ := k.Params.Get(ctx)
	k.clearVotesAndPreVotes(ctx, params.VotePeriod)
	k.updateWhitelist(ctx, params.Whitelist, whitelistedPairs)
	k.registerAbstainsByOmission(ctx, len(params.Whitelist), validatorPerformances)
	return validatorPerformances
}

// registerMissedVotes it parses all validators performance and increases the
// missed vote of those that did not vote.
func (k Keeper) registerMissedVotes(
	ctx sdk.Context,
	whitelistedPairs set.Set[asset.Pair],
	validatorPerformances types.ValidatorPerformances,
) {
	for _, validatorPerformance := range validatorPerformances {
		if int(validatorPerformance.MissCount) > 0 {
			k.MissCounters.Insert(
				ctx, validatorPerformance.ValAddress,
				k.MissCounters.GetOr(ctx, validatorPerformance.ValAddress, 0)+1)
			k.Logger(ctx).Info("vote miss", "validator", validatorPerformance.ValAddress.String())
		}
	}
}

func (k Keeper) registerAbstainsByOmission(
	ctx sdk.Context,
	numMarkets int,
	perfs types.ValidatorPerformances,
) {
	for valAddr, perf := range perfs {
		omitCount := int64(numMarkets) - (perf.WinCount + perf.AbstainCount + perf.MissCount)
		if omitCount > 0 {
			perf.AbstainCount += omitCount
			perfs[valAddr] = perf
		}
	}
}

// countVotesAndUpdateExchangeRates processes the votes and updates the
// ExchangeRates based on the results.
func (k Keeper) countVotesAndUpdateExchangeRates(
	ctx sdk.Context,
	pairVotes map[asset.Pair]types.ExchangeRateVotes,
	validatorPerformances types.ValidatorPerformances,
) {
	rewardBand := k.RewardBand(ctx)

	// Iterate through sorted keys for deterministic ordering.
	orderedPairVotes := omap.OrderedMap_Pair[types.ExchangeRateVotes](pairVotes)
	for pair := range orderedPairVotes.Range() {
		votes := pairVotes[pair]
		exchangeRate, _ := Tally(votes, rewardBand, validatorPerformances)

		k.SetPrice(ctx, pair, exchangeRate)

		_ = ctx.EventManager().EmitTypedEvent(&types.EventPriceUpdate{
			Pair:        pair.String(),
			Price:       exchangeRate,
			TimestampMs: ctx.BlockTime().UnixMilli(),
		})
	}
}

// getPairVotesAndWhitelistedPairs returns a map of pairs and votes excluding abstained votes
// and a set of all whitelisted pairs.
func (k Keeper) getPairVotesAndWhitelistedPairs(
	ctx sdk.Context,
	validatorPerformances types.ValidatorPerformances,
) (pairVotes map[asset.Pair]types.ExchangeRateVotes, whitelistedPairs set.Set[asset.Pair]) {
	pairVotes = k.groupVotesByPair(ctx, validatorPerformances)

	return k.removeInvalidVotes(ctx, pairVotes)
}

// resetExchangeRates removes all exchange rates from the state
// We remove the price for pair with expired prices or valid ballots
func (k Keeper) resetExchangeRates(ctx sdk.Context, pairVotes map[asset.Pair]types.ExchangeRateVotes) {
	params, _ := k.Params.Get(ctx)
	expirationBlocks := params.ExpirationBlocks

	for _, key := range k.ExchangeRates.Iterate(ctx, collections.Range[asset.Pair]{}).Keys() {
		_, isValid := pairVotes[key]
		exchangeRate, _ := k.ExchangeRates.Get(ctx, key)
		isExpired := exchangeRate.CreatedBlock+expirationBlocks <= uint64(ctx.BlockHeight())

		if isValid || isExpired {
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

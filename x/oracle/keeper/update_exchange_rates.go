package keeper

import (
	"sort"

	"github.com/NibiruChain/nibiru/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func (k Keeper) UpdateExchangeRates(ctx sdk.Context) {
	k.Logger(ctx).Info("processing validator price votes")
	params := k.GetParams(ctx)
	// Build claim map over all validators in active set
	validatorPerformanceMap := make(map[string]types.ValidatorPerformance)

	maxValidators := k.StakingKeeper.MaxValidators(ctx)
	iterator := k.StakingKeeper.ValidatorsPowerStoreIterator(ctx)
	defer iterator.Close()

	powerReduction := k.StakingKeeper.PowerReduction(ctx)

	i := 0
	for ; iterator.Valid() && i < int(maxValidators); iterator.Next() {
		validator := k.StakingKeeper.Validator(ctx, iterator.Value())

		// Exclude not bonded validator
		if validator.IsBonded() {
			valAddr := validator.GetOperator()
			validatorPerformanceMap[valAddr.String()] = types.NewValidatorPerformance(validator.GetConsensusPower(powerReduction), 0, 0, valAddr)
			i++
		}
	}

	pairsMap := make(map[string]struct{})
	for _, p := range k.Pairs.Iterate(ctx, collections.Range[string]{}).Keys() {
		pairsMap[p] = struct{}{}
	}

	for _, key := range k.ExchangeRates.Iterate(ctx, collections.Range[string]{}).Keys() {
		err := k.ExchangeRates.Delete(ctx, key)
		if err != nil {
			panic(err)
		}
	}
	// Organize votes to ballot by pair
	// NOTE: **Filter out inactive or jailed validators**
	// NOTE: **Make abstain votes to have zero vote power**
	pairBallotMap := k.OrganizeBallotByPair(ctx, validatorPerformanceMap)
	// remove ballots which are not passing
	RemoveInvalidBallots(ctx, k, pairsMap, pairBallotMap)
	// Iterate through ballots and update exchange rates; drop if not enough votes have been achieved.
	for pair, ballot := range pairBallotMap {
		sort.Sort(ballot)

		// Get weighted median of cross exchange rates
		exchangeRate := Tally(ctx, ballot, params.RewardBand, validatorPerformanceMap)

		// Set the exchange rate, emit ABCI event
		k.ExchangeRates.Insert(ctx, pair, exchangeRate)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(types.EventTypeExchangeRateUpdate,
				sdk.NewAttribute(types.AttributeKeyPair, pair),
				sdk.NewAttribute(types.AttributeKeyExchangeRate, exchangeRate.String()),
			),
		)
	}

	//---------------------------
	// Do miss counting & slashing
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

	// Distribute rewards to ballot winners
	k.RewardBallotWinners(ctx, pairsMap, validatorPerformanceMap)

	// Clear the ballot
	k.ClearBallots(ctx, params.VotePeriod)

	// Update vote targets
	k.ApplyWhitelist(ctx, params.Whitelist, pairsMap)
}

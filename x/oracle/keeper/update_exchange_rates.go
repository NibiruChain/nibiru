package keeper

import (
	"sort"

	gogotypes "github.com/gogo/protobuf/types"

	"github.com/NibiruChain/nibiru/collections/keys"

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
	for _, pair := range k.Pairs.Iterate(ctx, keys.NewRange[keys.StringKey]()).Keys() {
		pairsMap[string(pair)] = struct{}{}
	}

	for _, pair := range k.ExchangeRates.Iterate(ctx, keys.NewRange[keys.StringKey]()).Keys() {
		err := k.ExchangeRates.Delete(ctx, pair)
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
		k.SetExchangeRateWithEvent(ctx, pair, exchangeRate)
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
		valAddrKey := keys.String(claim.ValAddress.String())
		k.MissCounters.Insert(ctx,
			keys.String(valAddrKey),
			gogotypes.UInt64Value{
				// get from store, if it does not exist use 0 as default, and increase the result.
				Value: k.MissCounters.GetOr(ctx, keys.String(valAddrKey), gogotypes.UInt64Value{Value: 0}).Value + 1,
			},
		)
		k.Logger(ctx).Info("vote miss", "validator", claim.ValAddress.String())
	}

	// Distribute rewards to ballot winners
	k.RewardBallotWinners(ctx, pairsMap, validatorPerformanceMap)

	// Clear the ballot
	k.ClearBallots(ctx, params.VotePeriod)

	// Update vote targets
	k.ApplyWhitelist(ctx, params.Whitelist, pairsMap)
}

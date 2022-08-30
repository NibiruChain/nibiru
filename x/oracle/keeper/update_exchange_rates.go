package keeper

import (
	"sort"

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
	k.IteratePairs(ctx, func(pair string) bool {
		pairsMap[pair] = struct{}{}
		return false
	})

	k.ClearExchangeRates(ctx)
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
		k.SetMissCounter(ctx, claim.ValAddress, k.GetMissCounter(ctx, claim.ValAddress)+1)
		k.Logger(ctx).Info("vote miss", "validator", claim.ValAddress.String())
	}

	// Distribute rewards to ballot winners
	k.RewardBallotWinners(ctx, pairsMap, validatorPerformanceMap)

	// Clear the ballot
	k.ClearBallots(ctx, params.VotePeriod)

	// Update vote targets
	k.ApplyWhitelist(ctx, params.Whitelist, pairsMap)
}

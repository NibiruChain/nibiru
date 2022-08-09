package oracle

import (
	"time"

	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	"github.com/NibiruChain/nibiru/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker is called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	params := k.GetParams(ctx)
	if types.IsPeriodLastBlock(ctx, params.VotePeriod) {
		// Build claim map over all validators in active set
		validatorClaimMap := make(map[string]types.Claim)

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
				validatorClaimMap[valAddr.String()] = types.NewClaim(validator.GetConsensusPower(powerReduction), 0, 0, valAddr)
				i++
			}
		}

		// Pair-TobinTax map
		pairTobinTaxMap := make(map[string]sdk.Dec)
		k.IterateTobinTaxes(ctx, func(pair string, tobinTax sdk.Dec) bool {
			pairTobinTaxMap[pair] = tobinTax
			return false
		})

		// Clear all exchange rates
		k.IterateExchangeRates(ctx, func(pair string, _ sdk.Dec) (stop bool) {
			k.DeleteExchangeRate(ctx, pair)
			return false
		})

		// Organize votes to ballot by pair
		// NOTE: **Filter out inactive or jailed validators**
		// NOTE: **Make abstain votes to have zero vote power**
		pairBallotMap := k.OrganizeBallotByPair(ctx, validatorClaimMap)

		if referencePair := PickReferencePair(ctx, k, pairTobinTaxMap, pairBallotMap); referencePair != "" {
			// make voteMap of reference pair to calculate cross exchange rates
			referenceBallot := pairBallotMap[referencePair]
			referenceValidatorExchangeRateMap := referenceBallot.ToMap()
			referenceExchangeRate := referenceBallot.WeightedMedianWithAssertion()

			// Iterate through ballots and update exchange rates; drop if not enough votes have been achieved.
			for pair, ballot := range pairBallotMap {
				// Convert ballot to cross exchange rates
				if pair != referencePair {
					ballot = ballot.ToCrossRateWithSort(referenceValidatorExchangeRateMap)
				}

				// Get weighted median of cross exchange rates
				exchangeRate := Tally(ctx, ballot, params.RewardBand, validatorClaimMap)

				// Transform into the original exchange rate
				if pair != referencePair {
					exchangeRate = referenceExchangeRate.Quo(exchangeRate)
				}

				// Set the exchange rate, emit ABCI event
				k.SetExchangeRateWithEvent(ctx, pair, exchangeRate)
			}
		}

		//---------------------------
		// Do miss counting & slashing
		voteTargetsLen := len(pairTobinTaxMap)
		for _, claim := range validatorClaimMap {
			// Skip abstain & valid voters
			if int(claim.WinCount) == voteTargetsLen {
				continue
			}

			// Increase miss counter
			k.SetMissCounter(ctx, claim.Recipient, k.GetMissCounter(ctx, claim.Recipient)+1)
		}

		// Distribute rewards to ballot winners
		k.RewardBallotWinners(
			ctx,
			(int64)(params.VotePeriod),
			(int64)(params.RewardDistributionWindow),
			pairTobinTaxMap,
			validatorClaimMap,
		)

		// Clear the ballot
		k.ClearBallots(ctx, params.VotePeriod)

		// Update vote targets and tobin tax
		k.ApplyWhitelist(ctx, params.Whitelist, pairTobinTaxMap)
	}

	// Do slash who did miss voting over threshold and
	// reset miss counters of all validators at the last block of slash window
	if types.IsPeriodLastBlock(ctx, params.SlashWindow) {
		k.SlashAndResetMissCounters(ctx)
	}
}

package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestFuzz_Tally(t *testing.T) {
	validators := map[string]int64{}

	f := fuzz.New().NilChance(0).Funcs(
		func(e *sdk.Dec, c fuzz.Continue) {
			*e = sdk.NewDec(c.Int63())
		},
		func(e *map[string]int64, c fuzz.Continue) {
			numValidators := c.Intn(100) + 5

			for i := 0; i < numValidators; i++ {
				(*e)[sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()] = c.Int63n(100)
			}
		},
		func(e *map[string]types.ValidatorPerformance, c fuzz.Continue) {
			for validator, power := range validators {
				addr, err := sdk.ValAddressFromBech32(validator)
				require.NoError(t, err)
				(*e)[validator] = types.NewValidatorPerformance(power, addr)
			}
		},
		func(e *types.ExchangeRateBallot, c fuzz.Continue) {
			ballot := types.ExchangeRateBallot{}
			for addr, power := range validators {
				addr, _ := sdk.ValAddressFromBech32(addr)

				var rate sdk.Dec
				c.Fuzz(&rate)

				ballot = append(ballot, types.NewBallotVoteForTally(rate, c.RandString(), addr, power))
			}

			*e = ballot
		},
	)

	// set random pairs and validators
	f.Fuzz(&validators)

	claimMap := map[string]types.ValidatorPerformance{}
	f.Fuzz(&claimMap)

	ballot := types.ExchangeRateBallot{}
	f.Fuzz(&ballot)

	var rewardBand sdk.Dec
	f.Fuzz(&rewardBand)

	require.NotPanics(t, func() {
		Tally(ballot, rewardBand, claimMap)
	})
}

func TestFuzz_PickReferencePair(t *testing.T) {
	var pairs []string

	f := fuzz.New().NilChance(0).Funcs(
		func(e *[]string, c fuzz.Continue) {
			numPairs := c.Intn(100) + 5

			for i := 0; i < numPairs; i++ {
				*e = append(*e, testutil.RandStringBytes(5))
			}
		},
		func(e *sdk.Dec, c fuzz.Continue) {
			*e = sdk.NewDec(c.Int63())
		},
		func(e *map[string]sdk.Dec, c fuzz.Continue) {
			for _, pair := range pairs {
				var rate sdk.Dec
				c.Fuzz(&rate)

				(*e)[pair] = rate
			}
		},
		func(e *map[string]int64, c fuzz.Continue) {
			numValidator := c.Intn(100) + 5
			for i := 0; i < numValidator; i++ {
				(*e)[sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()] = int64(c.Intn(100) + 1)
			}
		},
		func(e *map[string]types.ExchangeRateBallot, c fuzz.Continue) {
			validators := map[string]int64{}
			c.Fuzz(&validators)

			for _, pair := range pairs {
				ballot := types.ExchangeRateBallot{}

				for addr, power := range validators {
					addr, _ := sdk.ValAddressFromBech32(addr)

					var rate sdk.Dec
					c.Fuzz(&rate)

					ballot = append(ballot, types.NewBallotVoteForTally(rate, pair, addr, power))
				}

				(*e)[pair] = ballot
			}
		},
	)

	// set random pairs
	f.Fuzz(&pairs)

	input, _ := setup(t)

	var panicAssertFn func(t assert.TestingT, f assert.PanicTestFunc, msgAndArgs ...interface{}) bool
	// Prevent collections error:
	// Panic value:	invalid StringKey bytes. StringKey must be at least length 2.

	// test OracleKeeper.Pairs.Insert
	voteTargets := map[string]struct{}{}
	f.Fuzz(&voteTargets)
	for key := range voteTargets {
		if len(key) == 1 {
			panicAssertFn = assert.Panics
		} else {
			panicAssertFn = assert.NotPanics
		}
		panicAssertFn(t, func() {
			input.OracleKeeper.Pairs.Insert(input.Ctx, key)
		}, "attempted to insert key: %s", key)
	}

	// test OracleKeeper.RemoveInvalidBallots
	voteMap := map[string]types.ExchangeRateBallot{}
	f.Fuzz(&voteMap)
	panicAssertFn = assert.NotPanics
	for k := range voteTargets {
		if len(k) == 1 {
			panicAssertFn = assert.Panics
		}
	}
	panicAssertFn(t, func() {
		input.OracleKeeper.RemoveInvalidBallots(input.Ctx, voteMap)
	}, "voteMap: %v", voteMap)
}

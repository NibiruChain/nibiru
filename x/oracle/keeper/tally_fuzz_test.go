package keeper_test

import (
	"sort"
	"testing"

	"github.com/cosmos/cosmos-sdk/x/staking"

	"github.com/NibiruChain/nibiru/x/oracle/keeper"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

var (
	stakingAmt = sdk.TokensFromConsensusPower(10, sdk.DefaultPowerReduction)

	randomExchangeRate = sdk.NewDec(1700)
)

func setup(t *testing.T) (keeper.TestInput, types.MsgServer) {
	input := keeper.CreateTestInput(t)
	params := input.OracleKeeper.GetParams(input.Ctx)
	params.VotePeriod = 1
	params.SlashWindow = 100
	input.OracleKeeper.SetParams(input.Ctx, params)
	h := keeper.NewMsgServerImpl(input.OracleKeeper)

	sh := staking.NewHandler(input.StakingKeeper)

	// Validator created
	_, err := sh(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[0], keeper.ValPubKeys[0], stakingAmt))
	require.NoError(t, err)
	_, err = sh(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[1], keeper.ValPubKeys[1], stakingAmt))
	require.NoError(t, err)
	_, err = sh(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[2], keeper.ValPubKeys[2], stakingAmt))
	require.NoError(t, err)
	staking.EndBlocker(input.Ctx, input.StakingKeeper)

	return input, h
}

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
				(*e)[validator] = types.NewValidatorPerformance(power, 0, 0, addr)
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

			sort.Sort(ballot)

			*e = ballot
		},
	)

	// set random pairs and validators
	f.Fuzz(&validators)

	input, _ := setup(t)

	claimMap := map[string]types.ValidatorPerformance{}
	f.Fuzz(&claimMap)

	ballot := types.ExchangeRateBallot{}
	f.Fuzz(&ballot)

	var rewardBand sdk.Dec
	f.Fuzz(&rewardBand)

	require.NotPanics(t, func() {
		keeper.Tally(input.Ctx, ballot, rewardBand, claimMap)
	})
}

func TestFuzz_PickReferencePair(t *testing.T) {
	var pairs []string

	f := fuzz.New().NilChance(0).Funcs(
		func(e *[]string, c fuzz.Continue) {
			numPairs := c.Intn(100) + 5

			for i := 0; i < numPairs; i++ {
				*e = append(*e, c.RandString())
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

				sort.Sort(ballot)
				(*e)[pair] = ballot
			}
		},
	)

	// set random pairs
	f.Fuzz(&pairs)

	input, _ := setup(t)

	voteTargets := map[string]struct{}{}
	f.Fuzz(&voteTargets)

	voteMap := map[string]types.ExchangeRateBallot{}
	f.Fuzz(&voteMap)

	require.NotPanics(t, func() {
		keeper.PickReferencePair(input.Ctx, input.OracleKeeper, voteTargets, voteMap)
	})
}

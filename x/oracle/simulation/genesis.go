package simulation

// DONTCOVER

import (
	sdkmath "cosmossdk.io/math"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// Simulation parameter constants
const (
	voteThresholdKey     = "vote_threshold"
	rewardBandKey        = "reward_band"
	slashFractionKey     = "slash_fraction"
	slashWindowKey       = "slash_window"
	minValidPerWindowKey = "min_valid_per_window"
)

// GenVotePeriod randomized VotePeriod
func GenVotePeriod(r *rand.Rand) uint64 {
	return uint64(1 + r.Intn(100))
}

// GenVoteThreshold randomized VoteThreshold
func GenVoteThreshold(r *rand.Rand) sdkmath.LegacyDec {
	return sdkmath.LegacyNewDecWithPrec(333, 3).Add(sdkmath.LegacyNewDecWithPrec(int64(r.Intn(333)), 3))
}

// GenRewardBand randomized RewardBand
func GenRewardBand(r *rand.Rand) sdkmath.LegacyDec {
	return sdkmath.LegacyZeroDec().Add(sdkmath.LegacyNewDecWithPrec(int64(r.Intn(100)), 3))
}

// GenRewardDistributionWindow randomized RewardDistributionWindow
func GenRewardDistributionWindow(r *rand.Rand) uint64 {
	return uint64(100 + r.Intn(100000))
}

// GenSlashFraction randomized SlashFraction
func GenSlashFraction(r *rand.Rand) sdkmath.LegacyDec {
	return sdkmath.LegacyZeroDec().Add(sdkmath.LegacyNewDecWithPrec(int64(r.Intn(100)), 3))
}

// GenSlashWindow randomized SlashWindow
func GenSlashWindow(r *rand.Rand) uint64 {
	return uint64(100 + r.Intn(100000))
}

// GenMinValidPerWindow randomized MinValidPerWindow
func GenMinValidPerWindow(r *rand.Rand) sdkmath.LegacyDec {
	return sdkmath.LegacyZeroDec().Add(sdkmath.LegacyNewDecWithPrec(int64(r.Intn(500)), 3))
}

// RandomizedGenState generates a random GenesisState for oracle
func RandomizedGenState(simState *module.SimulationState) {
	var voteThreshold sdkmath.LegacyDec
	simState.AppParams.GetOrGenerate(
		voteThresholdKey, &voteThreshold, simState.Rand,
		func(r *rand.Rand) { voteThreshold = GenVoteThreshold(r) },
	)

	var rewardBand sdkmath.LegacyDec
	simState.AppParams.GetOrGenerate(
		rewardBandKey, &rewardBand, simState.Rand,
		func(r *rand.Rand) { rewardBand = GenRewardBand(r) },
	)

	var slashFraction sdkmath.LegacyDec
	simState.AppParams.GetOrGenerate(
		slashFractionKey, &slashFraction, simState.Rand,
		func(r *rand.Rand) { slashFraction = GenSlashFraction(r) },
	)

	var slashWindow uint64
	simState.AppParams.GetOrGenerate(
		slashWindowKey, &slashWindow, simState.Rand,
		func(r *rand.Rand) { slashWindow = GenSlashWindow(r) },
	)

	var minValidPerWindow sdkmath.LegacyDec
	simState.AppParams.GetOrGenerate(
		minValidPerWindowKey, &minValidPerWindow, simState.Rand,
		func(r *rand.Rand) { minValidPerWindow = GenMinValidPerWindow(r) },
	)

	oracleGenesis := types.NewGenesisState(
		types.Params{
			VotePeriod:    uint64(10_000),
			VoteThreshold: voteThreshold,
			RewardBand:    rewardBand,
			Whitelist: []asset.Pair{
				asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				asset.Registry.Pair(denoms.USDC, denoms.NUSD),
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
			},
			SlashFraction:     slashFraction,
			SlashWindow:       slashWindow,
			MinValidPerWindow: minValidPerWindow,
		},
		[]types.ExchangeRateTuple{
			{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdkmath.LegacyNewDec(20_000)},
		},
		[]types.FeederDelegation{},
		[]types.MissCounter{},
		[]types.AggregateExchangeRatePrevote{},
		[]types.AggregateExchangeRateVote{},
		[]asset.Pair{},
		[]types.Rewards{},
	)

	bz, err := json.MarshalIndent(&oracleGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated oracle parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(oracleGenesis)
}

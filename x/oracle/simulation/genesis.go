package simulation

//DONTCOVER

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
func GenVoteThreshold(r *rand.Rand) sdk.Dec {
	return sdk.NewDecWithPrec(333, 3).Add(sdk.NewDecWithPrec(int64(r.Intn(333)), 3))
}

// GenRewardBand randomized RewardBand
func GenRewardBand(r *rand.Rand) sdk.Dec {
	return sdk.ZeroDec().Add(sdk.NewDecWithPrec(int64(r.Intn(100)), 3))
}

// GenRewardDistributionWindow randomized RewardDistributionWindow
func GenRewardDistributionWindow(r *rand.Rand) uint64 {
	return uint64(100 + r.Intn(100000))
}

// GenSlashFraction randomized SlashFraction
func GenSlashFraction(r *rand.Rand) sdk.Dec {
	return sdk.ZeroDec().Add(sdk.NewDecWithPrec(int64(r.Intn(100)), 3))
}

// GenSlashWindow randomized SlashWindow
func GenSlashWindow(r *rand.Rand) uint64 {
	return uint64(100 + r.Intn(100000))
}

// GenMinValidPerWindow randomized MinValidPerWindow
func GenMinValidPerWindow(r *rand.Rand) sdk.Dec {
	return sdk.ZeroDec().Add(sdk.NewDecWithPrec(int64(r.Intn(500)), 3))
}

// RandomizedGenState generates a random GenesisState for oracle
func RandomizedGenState(simState *module.SimulationState) {
	var voteThreshold sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, voteThresholdKey, &voteThreshold, simState.Rand,
		func(r *rand.Rand) { voteThreshold = GenVoteThreshold(r) },
	)

	var rewardBand sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, rewardBandKey, &rewardBand, simState.Rand,
		func(r *rand.Rand) { rewardBand = GenRewardBand(r) },
	)

	var slashFraction sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, slashFractionKey, &slashFraction, simState.Rand,
		func(r *rand.Rand) { slashFraction = GenSlashFraction(r) },
	)

	var slashWindow uint64
	simState.AppParams.GetOrGenerate(
		simState.Cdc, slashWindowKey, &slashWindow, simState.Rand,
		func(r *rand.Rand) { slashWindow = GenSlashWindow(r) },
	)

	var minValidPerWindow sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, minValidPerWindowKey, &minValidPerWindow, simState.Rand,
		func(r *rand.Rand) { minValidPerWindow = GenMinValidPerWindow(r) },
	)

	oracleGenesis := types.NewGenesisState(
		types.Params{
			VotePeriod:    uint64(10_000),
			VoteThreshold: voteThreshold,
			RewardBand:    rewardBand,
			Whitelist: []common.AssetPair{
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
			{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.NewDec(20_000)},
		},
		[]types.FeederDelegation{},
		[]types.MissCounter{},
		[]types.AggregateExchangeRatePrevote{},
		[]types.AggregateExchangeRateVote{},
		[]common.AssetPair{},
		[]types.PairReward{},
	)

	bz, err := json.MarshalIndent(&oracleGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated oracle parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(oracleGenesis)
}

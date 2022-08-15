package oracle

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// InitGenesis initialize default parameters
// and the keeper's address to pubkey map
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, data *types.GenesisState) {
	for _, d := range data.FeederDelegations {
		voter, err := sdk.ValAddressFromBech32(d.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		feeder, err := sdk.AccAddressFromBech32(d.FeederAddress)
		if err != nil {
			panic(err)
		}

		keeper.SetFeederDelegation(ctx, voter, feeder)
	}

	for _, ex := range data.ExchangeRates {
		keeper.SetExchangeRate(ctx, ex.Pair, ex.ExchangeRate)
	}

	for _, mc := range data.MissCounters {
		operator, err := sdk.ValAddressFromBech32(mc.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		keeper.SetMissCounter(ctx, operator, mc.MissCounter)
	}

	for _, ap := range data.AggregateExchangeRatePrevotes {
		valAddr, err := sdk.ValAddressFromBech32(ap.Voter)
		if err != nil {
			panic(err)
		}

		keeper.SetAggregateExchangeRatePrevote(ctx, valAddr, ap)
	}

	for _, av := range data.AggregateExchangeRateVotes {
		valAddr, err := sdk.ValAddressFromBech32(av.Voter)
		if err != nil {
			panic(err)
		}

		keeper.SetAggregateExchangeRateVote(ctx, valAddr, av)
	}

	if len(data.Pairs) > 0 {
		for _, tt := range data.Pairs {
			keeper.SetPair(ctx, tt.Name)
		}
	} else {
		for _, item := range data.Params.Whitelist {
			keeper.SetPair(ctx, item.Name)
		}
	}

	keeper.SetParams(ctx, data.Params)

	// check if the module account exists
	moduleAcc := keeper.GetOracleAccount(ctx)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}
}

// ExportGenesis writes the current store values
// to a genesis file, which can be imported again
// with InitGenesis
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) *types.GenesisState {
	params := keeper.GetParams(ctx)
	feederDelegations := []types.FeederDelegation{}
	keeper.IterateFeederDelegations(ctx, func(valAddr sdk.ValAddress, feederAddr sdk.AccAddress) (stop bool) {
		feederDelegations = append(feederDelegations, types.FeederDelegation{
			FeederAddress:    feederAddr.String(),
			ValidatorAddress: valAddr.String(),
		})
		return false
	})

	exchangeRates := []types.ExchangeRateTuple{}
	keeper.IterateExchangeRates(ctx, func(pair string, rate sdk.Dec) (stop bool) {
		exchangeRates = append(exchangeRates, types.ExchangeRateTuple{Pair: pair, ExchangeRate: rate})
		return false
	})

	missCounters := []types.MissCounter{}
	keeper.IterateMissCounters(ctx, func(operator sdk.ValAddress, missCounter uint64) (stop bool) {
		missCounters = append(missCounters, types.MissCounter{
			ValidatorAddress: operator.String(),
			MissCounter:      missCounter,
		})
		return false
	})

	aggregateExchangeRatePrevotes := []types.AggregateExchangeRatePrevote{}
	keeper.IterateAggregateExchangeRatePrevotes(ctx, func(_ sdk.ValAddress, aggregatePrevote types.AggregateExchangeRatePrevote) (stop bool) {
		aggregateExchangeRatePrevotes = append(aggregateExchangeRatePrevotes, aggregatePrevote)
		return false
	})

	aggregateExchangeRateVotes := []types.AggregateExchangeRateVote{}
	keeper.IterateAggregateExchangeRateVotes(ctx, func(_ sdk.ValAddress, aggregateVote types.AggregateExchangeRateVote) bool {
		aggregateExchangeRateVotes = append(aggregateExchangeRateVotes, aggregateVote)
		return false
	})

	pairs := []types.Pair{}
	keeper.IteratePairs(ctx, func(pair string) (stop bool) {
		pairs = append(pairs, types.Pair{Name: pair})
		return false
	})

	return types.NewGenesisState(params,
		exchangeRates,
		feederDelegations,
		missCounters,
		aggregateExchangeRatePrevotes,
		aggregateExchangeRateVotes,
		pairs)
}

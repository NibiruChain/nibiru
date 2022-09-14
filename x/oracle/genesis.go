package oracle

import (
	"fmt"

	gogotypes "github.com/gogo/protobuf/types"

	"github.com/NibiruChain/nibiru/collections/keys"

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

		keeper.FeederDelegations.Insert(ctx, keys.String(voter.String()), gogotypes.BytesValue{Value: feeder})
	}

	for _, ex := range data.ExchangeRates {
		keeper.ExchangeRates.Insert(ctx, keys.String(ex.Pair), sdk.DecProto{Dec: ex.ExchangeRate})
	}

	for _, mc := range data.MissCounters {
		operator, err := sdk.ValAddressFromBech32(mc.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		keeper.MissCounters.Insert(ctx, keys.String(operator.String()), gogotypes.UInt64Value{Value: mc.MissCounter})
	}

	for _, ap := range data.AggregateExchangeRatePrevotes {
		valAddr, err := sdk.ValAddressFromBech32(ap.Voter)
		if err != nil {
			panic(err)
		}
		keeper.Prevotes.Insert(ctx, keys.String(valAddr.String()), ap)
	}

	for _, av := range data.AggregateExchangeRateVotes {
		_, err := sdk.ValAddressFromBech32(av.Voter)
		if err != nil {
			panic(err)
		}

		keeper.Votes.Insert(ctx, keys.String(av.Voter), av)
	}

	if len(data.Pairs) > 0 {
		for _, tt := range data.Pairs {
			keeper.Pairs.Insert(ctx, keys.String(tt.Name))
		}
	} else {
		for _, item := range data.Params.Whitelist {
			keeper.Pairs.Insert(ctx, keys.String(item.Name))
		}
	}

	for _, pr := range data.PairRewards {
		keeper.SetPairReward(ctx, &pr)
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
	for _, v := range keeper.FeederDelegations.Iterate(ctx, keys.NewRange[keys.StringKey]()).KeyValues() {
		feederDelegations = append(feederDelegations, types.FeederDelegation{
			FeederAddress:    sdk.AccAddress(v.Value.Value).String(),
			ValidatorAddress: string(v.Key),
		})
	}

	exchangeRates := []types.ExchangeRateTuple{}
	for _, v := range keeper.ExchangeRates.Iterate(ctx, keys.NewRange[keys.StringKey]()).KeyValues() {
		exchangeRates = append(exchangeRates, types.ExchangeRateTuple{
			Pair:         string(v.Key),
			ExchangeRate: v.Value.Dec,
		})
	}

	missCounters := []types.MissCounter{}
	for _, v := range keeper.MissCounters.Iterate(ctx, keys.NewRange[keys.StringKey]()).KeyValues() {
		missCounters = append(missCounters, types.MissCounter{
			ValidatorAddress: string(v.Key),
			MissCounter:      v.Value.Value,
		})
	}

	aggregateExchangeRatePrevotes := keeper.Prevotes.Iterate(ctx, keys.NewRange[keys.StringKey]()).Values()

	aggregateExchangeRateVotes := keeper.Votes.Iterate(ctx, keys.NewRange[keys.StringKey]()).Values()

	pairs := []types.Pair{}
	for _, pair := range keeper.Pairs.Iterate(ctx, keys.NewRange[keys.StringKey]()).Keys() {
		pairs = append(pairs, types.Pair{Name: string(pair)})
	}

	pairRewards := []types.PairReward{}
	keeper.IterateAllPairRewards(ctx, func(rewards *types.PairReward) (stop bool) {
		pairRewards = append(pairRewards, *rewards)
		return false
	})

	return types.NewGenesisState(params,
		exchangeRates,
		feederDelegations,
		missCounters,
		aggregateExchangeRatePrevotes,
		aggregateExchangeRateVotes,
		pairs,
		pairRewards,
	)
}

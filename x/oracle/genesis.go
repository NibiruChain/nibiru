package oracle

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
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

		keeper.FeederDelegations.Insert(ctx, voter, feeder)
	}

	for _, ex := range data.ExchangeRates {
		keeper.SetPrice(ctx, ex.Pair, ex.ExchangeRate)
	}

	for _, missCounter := range data.MissCounters {
		operator, err := sdk.ValAddressFromBech32(missCounter.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		keeper.MissCounters.Insert(ctx, operator, missCounter.MissCounter)
	}

	for _, aggregatePrevote := range data.AggregateExchangeRatePrevotes {
		valAddr, err := sdk.ValAddressFromBech32(aggregatePrevote.Voter)
		if err != nil {
			panic(err)
		}

		keeper.Prevotes.Insert(ctx, valAddr, aggregatePrevote)
	}

	for _, aggregateVote := range data.AggregateExchangeRateVotes {
		valAddr, err := sdk.ValAddressFromBech32(aggregateVote.Voter)
		if err != nil {
			panic(err)
		}

		keeper.Votes.Insert(ctx, valAddr, aggregateVote)
	}

	if len(data.Pairs) > 0 {
		for _, tt := range data.Pairs {
			keeper.WhitelistedPairs.Insert(ctx, tt)
		}
	} else {
		for _, item := range data.Params.Whitelist {
			keeper.WhitelistedPairs.Insert(ctx, item)
		}
	}

	for _, pr := range data.PairRewards {
		keeper.PairRewards.Insert(ctx, pr.Id, pr)
	}

	// set last ID based on the last pair reward
	if len(data.PairRewards) != 0 {
		keeper.PairRewardsID.Set(ctx, data.PairRewards[len(data.PairRewards)-1].Id)
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
	for _, kv := range keeper.FeederDelegations.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
		feederDelegations = append(feederDelegations, types.FeederDelegation{
			FeederAddress:    kv.Value.String(),
			ValidatorAddress: kv.Key.String(),
		})
	}

	exchangeRates := []types.ExchangeRateTuple{}
	for _, er := range keeper.ExchangeRates.Iterate(ctx, collections.Range[asset.Pair]{}).KeyValues() {
		exchangeRates = append(exchangeRates, types.ExchangeRateTuple{Pair: er.Key, ExchangeRate: er.Value})
	}

	missCounters := []types.MissCounter{}
	for _, mc := range keeper.MissCounters.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
		missCounters = append(missCounters, types.MissCounter{
			ValidatorAddress: mc.Key.String(),
			MissCounter:      mc.Value,
		})
	}

	var pairs []asset.Pair
	pairs = append(pairs, keeper.WhitelistedPairs.Iterate(ctx, collections.Range[asset.Pair]{}).Keys()...)

	return types.NewGenesisState(params,
		exchangeRates,
		feederDelegations,
		missCounters,
		keeper.Prevotes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Values(),
		keeper.Votes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Values(),
		pairs,
		keeper.PairRewards.Iterate(ctx, collections.Range[uint64]{}).Values(),
	)
}

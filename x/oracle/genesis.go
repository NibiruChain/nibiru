package oracle

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"

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

		keeper.FeederDelegations.Set(ctx, voter, feeder)
	}

	for _, ex := range data.ExchangeRates {
		keeper.SetPrice(ctx, ex.Pair, ex.ExchangeRate)
	}

	for _, missCounter := range data.MissCounters {
		operator, err := sdk.ValAddressFromBech32(missCounter.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		keeper.MissCounters.Set(ctx, operator, missCounter.MissCounter)
	}

	for _, aggregatePrevote := range data.AggregateExchangeRatePrevotes {
		valAddr, err := sdk.ValAddressFromBech32(aggregatePrevote.Voter)
		if err != nil {
			panic(err)
		}

		keeper.Prevotes.Set(ctx, valAddr, aggregatePrevote)
	}

	for _, aggregateVote := range data.AggregateExchangeRateVotes {
		valAddr, err := sdk.ValAddressFromBech32(aggregateVote.Voter)
		if err != nil {
			panic(err)
		}

		keeper.Votes.Set(ctx, valAddr, aggregateVote)
	}

	if len(data.Pairs) > 0 {
		for _, tt := range data.Pairs {
			keeper.WhitelistedPairs.Set(ctx, tt)
		}
	} else {
		for _, item := range data.Params.Whitelist {
			keeper.WhitelistedPairs.Set(ctx, item)
		}
	}

	for _, pr := range data.Rewards {
		keeper.Rewards.Set(ctx, pr.Id, pr)
	}

	// set last ID based on the last pair reward
	if len(data.Rewards) != 0 {
		keeper.RewardsID.Set(ctx, data.Rewards[len(data.Rewards)-1].Id)
	}
	keeper.Params.Set(ctx, data.Params)

	// check if the module account exists
	moduleAcc := keeper.AccountKeeper.GetModuleAccount(ctx, types.ModuleName)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}
}

// ExportGenesis writes the current store values
// to a genesis file, which can be imported again
// with InitGenesis
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) *types.GenesisState {
	params, err := keeper.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	feederDelegations := []types.FeederDelegation{}
	iterFeederDelegations, err := keeper.FeederDelegations.Iterate(ctx, &collections.Range[sdk.ValAddress]{})
	if err != nil {
		keeper.Logger(ctx).Error("failed iterating feeder delegations", "error", err)
		return nil
	}
	kvFeederDelegations, err := iterFeederDelegations.KeyValues()
	if err != nil {
		keeper.Logger(ctx).Error("failed getting feeder delegations key values", "error", err)
		return nil
	}
	for _, kv := range kvFeederDelegations {
		feederDelegations = append(feederDelegations, types.FeederDelegation{
			FeederAddress:    kv.Value.String(),
			ValidatorAddress: kv.Key.String(),
		})
	}

	iterExchangeRates, err := keeper.ExchangeRates.Iterate(ctx, &collections.Range[asset.Pair]{})
	if err != nil {
		keeper.Logger(ctx).Error("failed iterating exchange rates", "error", err)
		return nil
	}
	kvExchangeRates, err := iterExchangeRates.KeyValues()
	if err != nil {
		keeper.Logger(ctx).Error("failed getting exchange rates key values", "error", err)
		return nil
	}
	exchangeRates := []types.ExchangeRateTuple{}
	for _, er := range kvExchangeRates {
		exchangeRates = append(exchangeRates, types.ExchangeRateTuple{Pair: er.Key, ExchangeRate: er.Value.ExchangeRate})
	}

	iterMissCounters, err := keeper.MissCounters.Iterate(ctx, &collections.Range[sdk.ValAddress]{})
	if err != nil {
		keeper.Logger(ctx).Error("failed iterating miss counters", "error", err)
		return nil
	}
	kvMissCounters, err := iterMissCounters.KeyValues()
	if err != nil {
		keeper.Logger(ctx).Error("failed getting miss counters values", "error", err)
		return nil
	}
	missCounters := []types.MissCounter{}
	for _, mc := range kvMissCounters {
		missCounters = append(missCounters, types.MissCounter{
			ValidatorAddress: mc.Key.String(),
			MissCounter:      mc.Value,
		})
	}

	var pairs []asset.Pair
	iterWhitelisterdPairs, err := keeper.WhitelistedPairs.Iterate(ctx, &collections.Range[asset.Pair]{})
	if err != nil {
		keeper.Logger(ctx).Error("failed iterating exchange rates", "error", err)
		return nil
	}
	keysWhitelisterdPairs, err := iterWhitelisterdPairs.Keys()
	if err != nil {
		keeper.Logger(ctx).Error("failed getting exchange rates key values", "error", err)
		return nil
	}

	pairs = append(pairs, keysWhitelisterdPairs...)

	iterPrevotes, err := keeper.Prevotes.Iterate(ctx, &collections.Range[sdk.ValAddress]{})
	if err != nil {
		keeper.Logger(ctx).Error("failed iterating prevotes", "error", err)
		return nil
	}
	valuesPrevotes, err := iterPrevotes.Values()
	if err != nil {
		keeper.Logger(ctx).Error("failed getting prevotes values", "error", err)
		return nil
	}

	iterVotes, err := keeper.Votes.Iterate(ctx, &collections.Range[sdk.ValAddress]{})
	if err != nil {
		keeper.Logger(ctx).Error("failed iterating votes", "error", err)
		return nil
	}
	valuesVotes, err := iterVotes.Values()
	if err != nil {
		keeper.Logger(ctx).Error("failed getting votes values", "error", err)
		return nil
	}

	iterRewards, err := keeper.Rewards.Iterate(ctx, &collections.Range[uint64]{})
	if err != nil {
		keeper.Logger(ctx).Error("failed iterating rewards", "error", err)
		return nil
	}
	valuesRewards, err := iterRewards.Values()
	if err != nil {
		keeper.Logger(ctx).Error("failed getting rewards values", "error", err)
		return nil
	}

	return types.NewGenesisState(
		params,
		exchangeRates,
		feederDelegations,
		missCounters,
		valuesPrevotes,
		valuesVotes,
		pairs,
		valuesRewards,
	)
}

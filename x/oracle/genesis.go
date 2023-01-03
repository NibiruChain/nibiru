package oracle

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

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

	for _, mc := range data.MissCounters {
		operator, err := sdk.ValAddressFromBech32(mc.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		keeper.MissCounters.Insert(ctx, operator, mc.MissCounter)
	}

	for _, ap := range data.AggregateExchangeRatePrevotes {
		valAddr, err := sdk.ValAddressFromBech32(ap.Voter)
		if err != nil {
			panic(err)
		}

		keeper.Prevotes.Insert(ctx, valAddr, ap)
	}

	for _, av := range data.AggregateExchangeRateVotes {
		valAddr, err := sdk.ValAddressFromBech32(av.Voter)
		if err != nil {
			panic(err)
		}

		keeper.Votes.Insert(ctx, valAddr, av)
	}

	if len(data.Pairs) > 0 {
		for _, tt := range data.Pairs {
			keeper.Pairs.Insert(ctx, tt)
		}
	} else {
		for _, item := range data.Params.Whitelist {
			keeper.Pairs.Insert(ctx, item)
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
	for _, er := range keeper.ExchangeRates.Iterate(ctx, collections.Range[string]{}).KeyValues() {
		exchangeRates = append(exchangeRates, types.ExchangeRateTuple{Pair: er.Key, ExchangeRate: er.Value})
	}

	missCounters := []types.MissCounter{}
	for _, mc := range keeper.MissCounters.Iterate(ctx, collections.Range[sdk.ValAddress]{}).KeyValues() {
		missCounters = append(missCounters, types.MissCounter{
			ValidatorAddress: mc.Key.String(),
			MissCounter:      mc.Value,
		})
	}

	var pairs []string
	pairs = append(pairs, keeper.Pairs.Iterate(ctx, collections.Range[string]{}).Keys()...)

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

package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

// NewGenesisState creates a new GenesisState object
func NewGenesisState(
	params Params, rates []ExchangeRateTuple,
	feederDelegations []FeederDelegation, missCounters []MissCounter,
	aggregateExchangeRatePrevotes []AggregateExchangeRatePrevote,
	aggregateExchangeRateVotes []AggregateExchangeRateVote,
	pairs []asset.Pair,
	pairRewards []Rewards,
) *GenesisState {
	return &GenesisState{
		Params:                        params,
		FeederDelegations:             feederDelegations,
		ExchangeRates:                 rates,
		MissCounters:                  missCounters,
		AggregateExchangeRatePrevotes: aggregateExchangeRatePrevotes,
		AggregateExchangeRateVotes:    aggregateExchangeRateVotes,
		Pairs:                         pairs,
		PairRewards:                   pairRewards,
	}
}

// DefaultGenesisState - default GenesisState
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(
		DefaultParams(),
		[]ExchangeRateTuple{},
		[]FeederDelegation{},
		[]MissCounter{},
		[]AggregateExchangeRatePrevote{},
		[]AggregateExchangeRateVote{},
		[]asset.Pair{},
		[]Rewards{})
}

// ValidateGenesis validates the oracle genesis state
func ValidateGenesis(data *GenesisState) error {
	return data.Params.Validate()
}

// GetGenesisStateFromAppState returns x/oracle GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}

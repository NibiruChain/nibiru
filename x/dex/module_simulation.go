package dex

import (
	"math/rand"

	dexsimulation "github.com/NibiruChain/nibiru/x/dex/simulation"
	"github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = dexsimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	// TODO: Determine the simulation weight value
	defaultWeight int = 100
)

// GenerateGenesisState creates a default GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	dexGenesis := types.DefaultGenesis()
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(dexGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	return []simtypes.ParamChange{}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	operations = append(operations, simulation.NewWeightedOperation(
		defaultWeight,
		dexsimulation.SimulateMsgCreatePool(am.accountKeeper, am.bankKeeper, am.keeper),
	), simulation.NewWeightedOperation(
		defaultWeight,
		dexsimulation.SimulateMsgSwap(am.accountKeeper, am.bankKeeper, am.keeper),
	), simulation.NewWeightedOperation(
		defaultWeight,
		dexsimulation.SimulateJoinPool(am.accountKeeper, am.bankKeeper, am.keeper),
	), simulation.NewWeightedOperation(
		defaultWeight,
		dexsimulation.SimulateExitPool(am.accountKeeper, am.bankKeeper, am.keeper),
	),
	)

	return operations
}

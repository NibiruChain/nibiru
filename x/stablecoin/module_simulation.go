package stablecoin

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	scsim "github.com/NibiruChain/nibiru/x/stablecoin/simulation"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

const (
	defaultWeightMsgMintStable int = 100
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	scGenesis := types.GenesisState{}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&scGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(
	simState module.SimulationState,
) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	opWeightMsgMintStable := "op_weight_msg_mint_sc"
	var weightMsgMintStable int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgMintStable, &weightMsgMintStable, nil,
		func(_ *rand.Rand) {
			weightMsgMintStable = defaultWeightMsgMintStable
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgMintStable,
		scsim.SimulateMsgMintStable(am.keeper, am.ak, am.bk),
	))

	return operations
}

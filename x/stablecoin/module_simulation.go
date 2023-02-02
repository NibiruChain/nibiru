package stablecoin

import (
	"math/rand"

	"github.com/NibiruChain/nibiru/x/common/testutil"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	scsim "github.com/NibiruChain/nibiru/x/stablecoin/simulation"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

// avoid unused import issue
var (
	_ = testutil.AccAddress
	_ = scsim.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
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

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	return []simtypes.ParamChange{}
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

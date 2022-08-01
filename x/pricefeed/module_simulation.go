package pricefeed

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/NibiruChain/nibiru/x/common"
	pricefeedsimulation "github.com/NibiruChain/nibiru/x/pricefeed/simulation"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgPostPrice = "op_weight_msg_create_chain"
	// TODO: Determine the simulation weight value
	defaultWeightMsgPostPrice int = 100
	AssetPairsKey                 = "Pairs"
	TwapLookbackWindow            = "TwapLookbackWindow"
	maxAssetPairs                 = 100
	maxPostedPrices               = 100
	maxLookbackWindowMinutes      = 7 * 24 * 60
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	var (
		assetPairs         common.AssetPairs
		twapLookbackWindow time.Duration
	)
	simState.AppParams.GetOrGenerate(simState.Cdc, AssetPairsKey, &assetPairs, simState.Rand,
		func(r *rand.Rand) { assetPairs = genAssetPairs(r) },
	)
	simState.AppParams.GetOrGenerate(simState.Cdc, TwapLookbackWindow, &twapLookbackWindow, simState.Rand,
		func(r *rand.Rand) { twapLookbackWindow = genTwapLookbackWindow(r) },
	)
	oracles := make([]string, len(simState.Accounts))
	for i := range oracles {
		oracles[i] = simState.Accounts[i].Address.String()
	}
	r := simState.Rand
	postedPrices := make([]types.PostedPrice, r.Intn(maxPostedPrices))

	for i := range postedPrices {
		if len(assetPairs) == 0 || len(oracles) == 0 {
			continue
		}
		postedPrices[i] = types.PostedPrice{
			PairID: assetPairs[r.Intn(len(assetPairs)-1)].String(),
			Oracle: oracles[r.Intn(len(oracles)-1)],
			Expiry: pricefeedsimulation.GenFutureDate(r, simState.GenTimestamp),
			Price:  pricefeedsimulation.GenPrice(r),
		}
	}
	pricefeedGenesis := types.GenesisState{
		Params: types.Params{
			Pairs:              assetPairs,
			TwapLookbackWindow: twapLookbackWindow,
		},
		PostedPrices:   postedPrices,
		GenesisOracles: oracles,
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&pricefeedGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, "Pairs", func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(genAssetPairs(r)))
		}),
		simulation.NewSimParamChange(types.ModuleName, "TwapLookbackWindow", func(r *rand.Rand) string {
			return fmt.Sprintf("\"%d\"", genTwapLookbackWindow(r))
		}),
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgPostPrice int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgPostPrice, &weightMsgPostPrice, nil,
		func(_ *rand.Rand) {
			weightMsgPostPrice = defaultWeightMsgPostPrice
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgPostPrice,
		pricefeedsimulation.SimulateMsgPostPrice(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	return operations
}

func genAssetPairs(r *rand.Rand) common.AssetPairs {
	assetPairs := make(common.AssetPairs, r.Intn(maxAssetPairs))
	for i := range assetPairs {
		a1, a2 := simtypes.RandStringOfLength(r, 5), simtypes.RandStringOfLength(r, 5)
		pair := common.MustNewAssetPair(fmt.Sprintf("%s:%s", a1, a2))
		assetPairs[i] = pair
	}
	return assetPairs
}

func genTwapLookbackWindow(r *rand.Rand) time.Duration {
	return time.Duration(r.Intn(maxLookbackWindowMinutes)) * time.Minute
}

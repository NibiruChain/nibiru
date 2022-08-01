package simulation

import (
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

func GenerateGenesis(simState *module.SimulationState) {
	var (
		assetPairs         common.AssetPairs
		twapLookbackWindow time.Duration
	)
	simState.AppParams.GetOrGenerate(simState.Cdc, AssetPairsKey, &assetPairs, simState.Rand,
		func(r *rand.Rand) { assetPairs = genAssetPairs(r) },
	)
	simState.AppParams.GetOrGenerate(simState.Cdc, TwapLookbackWindowKey, &twapLookbackWindow, simState.Rand,
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
			Expiry: GenFutureDate(r, simState.GenTimestamp),
			Price:  GenPrice(r),
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

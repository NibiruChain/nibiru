package simulation

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// RandomizedGenState generates a random GenesisState for the perp module
func RandomizedGenState(simState *module.SimulationState) {
	// var tradeLimitRatio sdk.Dec

	vpoolGenesis := types.GenesisState{
		Vpools: []types.VPool{
			{
				Pair:                   common.Pair_BTC_NUSD,
				TradeLimitRatio:        sdk.OneDec(),
				QuoteAssetReserve:      sdk.NewDec(10e12),
				BaseAssetReserve:       sdk.NewDec(10e12),
				FluctuationLimitRatio:  sdk.OneDec(),
				MaxOracleSpreadRatio:   sdk.OneDec(),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.NewDec(10),
			},
		},
	}

	vpoolGenesisBytes, err := json.MarshalIndent(&vpoolGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated vpools:\n%s\n", vpoolGenesisBytes)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&vpoolGenesis)

	// pricefeedGenesis := pricefeedtypes.DefaultGenesis()
	// pricefeedGenesisBytes, err := json.MarshalIndent(&pricefeedGenesis, "", " ")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("Selected randomly generated pricefeed genesis:\n%s\n", pricefeedGenesisBytes)
	// simState.GenState[pricefeedtypes.ModuleName] = simState.Cdc.MustMarshalJSON(pricefeedGenesis)
}

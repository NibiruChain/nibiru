package simulation

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

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
				QuoteAssetReserve:      sdk.NewDec(10e6).Add(simtypes.RandomDecAmount(simState.Rand, sdk.NewDec(10e6))),
				BaseAssetReserve:       sdk.NewDec(10e6).Add(simtypes.RandomDecAmount(simState.Rand, sdk.NewDec(10e6))),
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
}

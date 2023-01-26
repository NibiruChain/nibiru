package simulation

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// RandomizedGenState generates a random GenesisState for the perp module
func RandomizedGenState(simState *module.SimulationState) {
	// var tradeLimitRatio sdk.Dec
	smallDec := sdk.MustNewDecFromStr("0.0001")

	maxLeverage := sdk.OneDec().Add(simtypes.RandomDecAmount(simState.Rand, sdk.MustNewDecFromStr("15")))
	maintenanceMarginRatio := sdk.MaxDec(smallDec, simtypes.RandomDecAmount(simState.Rand, sdk.OneDec().Quo(maxLeverage)))

	vpoolGenesis := types.GenesisState{
		Vpools: []types.Vpool{
			{
				Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				QuoteAssetReserve: sdk.NewDec(10e12).Add(simtypes.RandomDecAmount(simState.Rand, sdk.NewDec(10e12))),
				BaseAssetReserve:  sdk.NewDec(10e12).Add(simtypes.RandomDecAmount(simState.Rand, sdk.NewDec(10e12))),
				Config: types.VpoolConfig{
					FluctuationLimitRatio:  sdk.MaxDec(smallDec, simtypes.RandomDecAmount(simState.Rand, sdk.OneDec())),
					MaintenanceMarginRatio: maintenanceMarginRatio,
					MaxLeverage:            maxLeverage,
					MaxOracleSpreadRatio:   sdk.MaxDec(smallDec, simtypes.RandomDecAmount(simState.Rand, sdk.OneDec())),
					TradeLimitRatio:        sdk.MaxDec(smallDec, simtypes.RandomDecAmount(simState.Rand, sdk.OneDec())),
				},
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

package simulation

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

// RandomizedGenState generates a random GenesisState for the perp module
func RandomizedGenState(simState *module.SimulationState) {
	vpoolGenesis := vpooltypes.GenesisState{
		Vpools: []*vpooltypes.VPool{
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

	vpools, err := json.MarshalIndent(&vpoolGenesis.Vpools, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated vpools:\n%s\n", vpools)
	simState.GenState[vpooltypes.ModuleName] = simState.Cdc.MustMarshalJSON(&vpoolGenesis)

	pricefeedGenesis := pricefeedtypes.DefaultGenesis()

	pricefeedGenesisBytes, err := json.MarshalIndent(&pricefeedGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated pricefeed genesis:\n%s\n", pricefeedGenesisBytes)
	simState.GenState[pricefeedtypes.ModuleName] = simState.Cdc.MustMarshalJSON(pricefeedGenesis)

	perpGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		PairMetadata: []types.PairMetadata{
			{
				Pair:                       common.Pair_BTC_NUSD,
				CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
			},
		},
	}
	perpGenesisBytes, err := json.MarshalIndent(&perpGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated perp genesis:\n%s\n", perpGenesisBytes)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&perpGenesis)
}

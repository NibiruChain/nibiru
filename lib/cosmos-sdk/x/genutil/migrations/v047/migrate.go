package v047

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	bankv4 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/migrations/v4"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	v1distr "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution/migrations/v1"
	v3distr "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution/migrations/v3"
	distrtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/genutil/types"
	v4gov "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/migrations/v4"
	govv1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"
)

// Migrate migrates exported state from v0.46 to a v0.47 genesis state.
func Migrate(appState types.AppMap, clientCtx client.Context) types.AppMap {
	// Migrate x/bank.
	bankState := appState[banktypes.ModuleName]
	if len(bankState) > 0 {
		var oldBankState banktypes.GenesisState
		clientCtx.Codec.MustUnmarshalJSON(bankState, &oldBankState)
		newBankState := bankv4.MigrateGenState(&oldBankState)
		appState[banktypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(newBankState)
	}

	if govOldState, ok := appState[v4gov.ModuleName]; ok {
		// unmarshal relative source genesis application state
		var old govv1.GenesisState
		clientCtx.Codec.MustUnmarshalJSON(govOldState, &old)

		// delete deprecated x/gov genesis state
		delete(appState, v4gov.ModuleName)

		// set the x/gov genesis state with new state.
		new, err := v4gov.MigrateJSON(&old)
		if err != nil {
			panic(err)
		}
		appState[v4gov.ModuleName] = clientCtx.Codec.MustMarshalJSON(new)
	}

	// Migrate x/distribution params (reset unused)
	if oldDistState, ok := appState[v1distr.ModuleName]; ok {
		var old distrtypes.GenesisState
		clientCtx.Codec.MustUnmarshalJSON(oldDistState, &old)
		newDistState := v3distr.MigrateJSON(&old)
		appState[v1distr.ModuleName] = clientCtx.Codec.MustMarshalJSON(newDistState)
	}

	return appState
}

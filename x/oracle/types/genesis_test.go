package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestGenesisValidation(t *testing.T) {
	genState := types.DefaultGenesisState()
	require.NoError(t, types.ValidateGenesis(genState))

	genState.Params.VotePeriod = 0
	require.Error(t, types.ValidateGenesis(genState))
}

func TestGetGenesisStateFromAppState(t *testing.T) {
	cdc := app.MakeEncodingConfig().Codec
	appState := make(map[string]json.RawMessage)

	defaultGenesisState := types.DefaultGenesisState()
	appState[types.ModuleName] = cdc.MustMarshalJSON(defaultGenesisState)
	require.Equal(t, *defaultGenesisState, *types.GetGenesisStateFromAppState(cdc, appState))
}

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

func TestReadSavedRPCBlockResults(t *testing.T) {
	event := abci.Event(sdk.NewEvent("usdc_migration", sdk.NewAttribute("upgrade", "v2.23.0"),
		sdk.NewAttribute("height", "100"), sdk.NewAttribute("status", "skipped"),
		sdk.NewAttribute("reason", "registry_not_deployed")))
	for _, height := range []any{"100", 100} {
		data, err := json.Marshal(map[string]any{"result": map[string]any{"height": height, "begin_block_events": []abci.Event{event}}})
		require.NoError(t, err)
		var output bytes.Buffer
		require.NoError(t, run(bytes.NewReader(data), &output))
		require.Contains(t, output.String(), `"status": "skipped"`)
		require.Contains(t, output.String(), `"collected_base_units": "0"`)
	}
}

func TestRejectRPCFailureAndWrongBlock(t *testing.T) {
	for _, input := range []string{
		`{"error":{"message":"height not available"}}`,
		`{"result":{"height":"1","begin_block_events":[]}}`,
		`{"result":{"height":"2","begin_block_events":[{"type":"usdc_migration","attributes":[{"key":"upgrade","value":"v2.23.0"},{"key":"height","value":"1"},{"key":"status","value":"skipped"}]}]}}`,
	} {
		var output bytes.Buffer
		require.Error(t, run(strings.NewReader(input), &output))
		require.Empty(t, output.String())
	}
}

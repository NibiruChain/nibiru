package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	devgas "github.com/NibiruChain/nibiru/v2/x/devgas/v1"
	"github.com/NibiruChain/nibiru/v2/x/devgas/v1/client/cli"
)

// TestNoTxCommands checks that module x/devgas adds nothing under `tx`.
func TestNoTxCommands(t *testing.T) {
	require.Nil(t, devgas.AppModuleBasic{}.GetTxCmd())
}

// TestQueryCommands checks that the read-only queries remain under
// `query devgas`.
func TestQueryCommands(t *testing.T) {
	cmd := cli.GetQueryCmd()
	require.Equal(t, "devgas", cmd.Name())

	var names []string
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	require.ElementsMatch(t, []string{
		"contracts", "contract", "params", "withdrawer-contracts",
	}, names)
}

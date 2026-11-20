package impl_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	nibidcmd "github.com/NibiruChain/nibiru/v2/x/cli/impl"
)

func helpOf(t *testing.T, cmd *cobra.Command) string {
	t.Helper()
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"--help"})
	require.NoError(t, cmd.Execute())
	return out.String()
}

func TestTxHelpGroups(t *testing.T) {
	cmd := nibidcmd.TxCmd()
	for _, c := range cmd.Commands() {
		require.NotEmpty(t, c.GroupID, c.Name())
	}
	help := helpOf(t, cmd)
	for _, section := range []string{
		"Everyday Modules:", "Offline Signing:", "Other Modules:",
	} {
		require.Contains(t, help, section)
	}
	// No command moved or disappeared.
	for _, name := range []string{
		"bank", "wasm", "evm", "sign", "sign-batch", "multi-sign",
		"multisign-batch", "validate-signatures", "encode", "decode",
		"broadcast", "tokenfactory", "ibc-transfer",
	} {
		_, _, err := cmd.Find([]string{name})
		require.NoError(t, err, name)
	}
}

func TestQueryHelpGroups(t *testing.T) {
	cmd := nibidcmd.QueryCmd()
	for _, c := range cmd.Commands() {
		require.NotEmpty(t, c.GroupID, c.Name())
	}
	help := helpOf(t, cmd)
	for _, section := range []string{
		"Everyday Modules:", "Chain Data:", "Other Modules:",
	} {
		require.Contains(t, help, section)
	}
}

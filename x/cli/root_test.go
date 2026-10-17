package main_test

import (
	"bytes"
	"testing"

	// Nibiru
	"github.com/NibiruChain/nibiru/v2/app"
	nibid "github.com/NibiruChain/nibiru/v2/x/cli"

	// Cosmos-SDK
	svrcmd "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/cmd"

	// Tendermint
	"github.com/stretchr/testify/require"
)

func TestRootCmdConfig(t *testing.T) {
	home := t.TempDir()
	rootCmd, _ := nibid.NewRootCmd()
	rootCmd.SetArgs([]string{
		"config",
		"query-mode",
		"direct",
		"--home",
		home,
	})
	require.NoError(t, svrcmd.Execute(rootCmd, "", app.DefaultNodeHome))

	rootCmd, _ = nibid.NewRootCmd()
	output := bytes.NewBuffer(nil)
	rootCmd.SetOut(output)
	rootCmd.SetArgs([]string{
		"config",
		"query-mode",
		"--home",
		home,
	})

	require.Contains(t, rootCmd.Aliases, "nibiru")

	require.NoError(t, svrcmd.Execute(rootCmd, "", app.DefaultNodeHome))
	require.Equal(t, "direct\n", output.String())
}

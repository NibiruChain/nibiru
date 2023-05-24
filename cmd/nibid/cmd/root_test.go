package cmd_test

import (
	"testing"

	// Nibiru
	"github.com/NibiruChain/nibiru/app"
	nibid "github.com/NibiruChain/nibiru/cmd/nibid/cmd"

	// Cosmos-SDK
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	// Tendermint
	"github.com/stretchr/testify/require"
)

func TestRootCmdConfig(t *testing.T) {
	rootCmd, _ := nibid.NewRootCmd()
	cmds := []string{
		"config",
	}
	rootCmd.SetArgs(cmds)

	require.NoError(t, svrcmd.Execute(rootCmd, "", app.DefaultNodeHome))
}

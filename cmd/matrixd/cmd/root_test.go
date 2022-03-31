package cmd

import (
	"testing"

	// Matrix
	"github.com/MatrixDao/matrix/app"
	// matrixd "github.com/MatrixDao/matrix/cmd/matrixd/cmd"

	// Cosmos-SDK
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	// Tendermint
	"github.com/stretchr/testify/require"
)

func TestRootCmdConfig(t *testing.T) {
	rootCmd, _ := NewRootCmd()
	cmds := []string{
		// "config",
		// "query",
		// "tx",
	}
	rootCmd.SetArgs(cmds)

	require.NoError(t, svrcmd.Execute(rootCmd, app.DefaultNodeHome))

}

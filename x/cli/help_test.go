package main_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	nibid "github.com/NibiruChain/nibiru/v2/x/cli"
)

var hiddenGlobalFlags = []string{
	"--log_format", "--log_level", "--log_no_color", "--trace",
}

func TestRootHelpShape(t *testing.T) {
	out, err := runRoot(t, "", "--help")
	require.NoError(t, err)

	require.NotContains(t, out, "Aliases:")
	require.NotContains(t, out, "Usage:\n  nibid [command]")
	require.Contains(t, out, "Examples:\n")
	require.Contains(t, out, nibid.RootHelpFooter)

	// Only these section headings appear.
	headings := regexp.MustCompile(`(?m)^[A-Z][A-Za-z ]*:$`).FindAllString(out, -1)
	require.Equal(t, []string{"Examples:", "Commands:", "Flags:"}, headings)
	require.NotContains(t, out, "Everyday Commands")
	require.NotContains(t, out, "Run a Node")

	// Rows come in the deliberate order, not alphabetical.
	rows := []string{}
	inCommands := false
	for _, line := range strings.Split(out, "\n") {
		switch {
		case line == "Commands:":
			inCommands = true
		case inCommands && strings.HasPrefix(line, "  "):
			rows = append(rows, strings.Fields(line)[0])
		case inCommands:
			inCommands = false
		}
	}
	require.Equal(t, []string{
		"tx", "q", "keys", "config", "status", "debug", "localnet", "node",
		"help", "version",
	}, rows)
	require.Contains(t, out, "Query the chain (alias: query)")

	// Hidden legacy commands and moved commands stay out of the list.
	for _, name := range []string{
		"start", "init", "tendermint", "base64-decode", "build-stargate-wasm",
	} {
		require.NotContains(t, rows, name)
	}
}

func TestRootExamplesResolve(t *testing.T) {
	rootCmd, _ := nibid.NewRootCmd()
	out, err := runRoot(t, "", "--help")
	require.NoError(t, err)

	for _, ex := range nibid.RootExamples {
		require.Contains(t, out, ex.Command)

		words := strings.Fields(strings.SplitN(ex.Command, "|", 2)[0])
		require.Equal(t, "nibid", words[0], ex.Command)
		path := []string{}
		for _, w := range words[1:] {
			if strings.HasPrefix(w, "<") || strings.HasPrefix(w, "'") ||
				strings.HasPrefix(w, "-") || strings.HasSuffix(w, "unibi") {
				break
			}
			path = append(path, w)
		}
		cmd, _, err := rootCmd.Find(path)
		require.NoError(t, err, ex.Command)
		require.NotEqual(t, rootCmd, cmd, "%q resolved to the root", ex.Command)
		// Words after the resolved command are its arguments (for example
		// `config node <rpc-url>`), so the command must be a leaf.
		resolved := len(strings.Fields(cmd.CommandPath())) - 1
		if resolved < len(path) {
			require.False(t, cmd.HasSubCommands(),
				"%q: %v did not resolve fully", ex.Command, path)
		}
	}
}

func TestQueryAndAliasResolve(t *testing.T) {
	rootCmd, _ := nibid.NewRootCmd()
	for _, name := range []string{"q", "query"} {
		cmd, _, err := rootCmd.Find([]string{name, "bank", "balances"})
		require.NoError(t, err)
		require.Equal(t, "balances", cmd.Name())
	}
}

func TestGlobalFlagsHiddenUntilHelpVerbose(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"keys", "list", "--help"},
		{"tx", "bank", "send", "--help"},
	} {
		out, err := runRoot(t, "", args...)
		require.NoError(t, err, args)
		for _, f := range hiddenGlobalFlags {
			require.NotContains(t, out, f, "%v", args)
		}
		require.Contains(t, out, "--home", "%v", args)
	}

	for _, args := range [][]string{
		{"--help-verbose"},
		{"keys", "list", "--help-verbose"},
		{"node", "start", "--help-verbose"},
		{"tx", "bank", "send", "--help", "--help-verbose"},
	} {
		out, err := runRoot(t, "", args...)
		require.NoError(t, err, args)
		for _, f := range hiddenGlobalFlags {
			require.Contains(t, out, f, "%v", args)
		}
	}
}

// Hidden flags still parse, and --home stays visible.
func TestHiddenGlobalFlagsStillParse(t *testing.T) {
	out, err := runRoot(t, "", "config", "query-mode",
		"--log_level", "debug", "--log_format", "json", "--log_no_color",
		"--trace")
	require.NoError(t, err, out)
}

package main_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
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
	require.Contains(t, out, nibid.RootHelpFooter("nibiru"))

	// Only these section headings appear.
	headings := regexp.MustCompile(`(?m)^[A-Z][A-Za-z ]*:$`).FindAllString(out, -1)
	require.Equal(t, []string{"Commands:", "Flags:"}, headings)
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

func TestInvokedName(t *testing.T) {
	for arg0, want := range map[string]string{
		"nibid":                 "nibid",
		"/home/u/go/bin/nibid":  "nibid",
		"nibiru":                "nibiru",
		"/usr/local/bin/nibiru": "nibiru",
		"./build/nibiru":        "nibiru",
		"nibiru.exe":            "nibiru",
		"main.test":             "nibiru",
		"my-copy":               "nibiru",
		"":                      "nibiru",
	} {
		require.Equal(t, want, nibid.InvokedName(arg0), arg0)
	}
}

// Help prints the invoked name in the usage line of every command, the flag
// descriptions, and the closing hint. The default home stays ~/.nibid.
func TestHelpPrintsInvokedName(t *testing.T) {
	for _, name := range []string{"nibiru", "nibid"} {
		t.Run(name, func(t *testing.T) {
			other := map[string]string{"nibiru": "nibid", "nibid": "nibiru"}[name]
			rootCmd, _ := nibid.NewRootCmdNamed(name)
			out := new(bytes.Buffer)
			rootCmd.SetOut(out)
			rootCmd.SetArgs([]string{"--help"})
			require.NoError(t, rootCmd.Execute())
			help := out.String()

			require.Contains(t, help, "help for "+name)
			require.Contains(t, help, nibid.RootHelpFooter(name))
			require.NotContains(t, help, `"`+other+" <command>")
			// On-disk defaults do not follow the invoked name.
			require.True(t, strings.HasSuffix(app.DefaultNodeHome, "/.nibid"))

			rootCmd, _ = nibid.NewRootCmdNamed(name)
			out.Reset()
			rootCmd.SetOut(out)
			rootCmd.SetArgs([]string{"keys", "list", "--help"})
			require.NoError(t, rootCmd.Execute())
			require.Contains(t, out.String(), name+" keys list")
			require.NotContains(t, out.String(), other+" keys list")
		})
	}
}

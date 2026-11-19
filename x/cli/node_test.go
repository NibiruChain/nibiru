package main_test

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	cmtcli "github.com/NibiruChain/nibiru/v2/bft/libs/cli"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	nibid "github.com/NibiruChain/nibiru/v2/x/cli"
)

// nodeCommandNames are the legacy top-level names that moved under `node`.
var nodeCommandNames = []string{
	"start", "init", "export", "genesis", "prune", "rollback", "tendermint",
	"testnet", "evm-tx-index",
}

func flagNames(c *cobra.Command) []string {
	names := []string{}
	c.LocalFlags().VisitAll(func(f *pflag.Flag) { names = append(names, f.Name) })
	sort.Strings(names)
	return names
}

func subcommandNames(c *cobra.Command) []string {
	names := []string{}
	for _, sub := range c.Commands() {
		names = append(names, sub.Name())
	}
	sort.Strings(names)
	return names
}

// walkPair asserts that the legacy command and the node command are
// equivalent, recursing into subcommands.
func walkPair(t *testing.T, path string, legacy, moved *cobra.Command) {
	t.Helper()
	require.NotSame(t, legacy, moved, path)
	require.Equal(t, legacy.Use, moved.Use, path)
	require.Equal(t, legacy.Short, moved.Short, path)
	require.Equal(t, legacy.Aliases, moved.Aliases, path)
	require.Equal(t, flagNames(legacy), flagNames(moved), path)
	require.Equal(t, subcommandNames(legacy), subcommandNames(moved), path)
	for _, sub := range legacy.Commands() {
		movedSub, _, err := moved.Find([]string{sub.Name()})
		require.NoError(t, err, path+" "+sub.Name())
		require.Equal(t, sub.Name(), movedSub.Name(), path+" "+sub.Name())
		walkPair(t, path+" "+sub.Name(), sub, movedSub)
	}
}

func TestNodeCommandTree(t *testing.T) {
	rootCmd, _ := nibid.NewRootCmd()

	nodeCmd, _, err := rootCmd.Find([]string{"node"})
	require.NoError(t, err)
	require.Equal(t, "node", nodeCmd.Name())
	require.False(t, nodeCmd.Hidden)

	for _, name := range nodeCommandNames {
		t.Run(name, func(t *testing.T) {
			legacy, rest, err := rootCmd.Find([]string{name})
			require.NoError(t, err)
			require.Empty(t, rest)
			require.Equal(t, name, legacy.Name())
			require.Equal(t, rootCmd, legacy.Parent())
			require.True(t, legacy.Hidden, "legacy %s must be hidden", name)

			moved, rest, err := rootCmd.Find([]string{"node", name})
			require.NoError(t, err)
			require.Empty(t, rest)
			require.Equal(t, nodeCmd, moved.Parent())
			require.False(t, moved.Hidden)

			walkPair(t, name, legacy, moved)
		})
	}

	// `comet` aliases `tendermint` in both places.
	for _, path := range [][]string{
		{"comet", "show-node-id"}, {"node", "comet", "show-node-id"},
		{"node", "tendermint", "show-node-id"},
	} {
		c, _, err := rootCmd.Find(path)
		require.NoError(t, err, path)
		require.Equal(t, "show-node-id", c.Name())
	}
}

func TestNodeHelp(t *testing.T) {
	rootCmd, _ := nibid.NewRootCmd()
	out := new(bytes.Buffer)
	rootCmd.SetOut(out)
	rootCmd.SetArgs([]string{"--help"})
	require.NoError(t, rootCmd.Execute())

	help := out.String()
	require.Contains(t, help, "Everyday Commands:")
	require.Contains(t, help, "Run a Node:")
	lines := map[string]bool{}
	for _, line := range strings.Split(help, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.HasPrefix(line, "  ") {
			lines[fields[0]] = true
		}
	}
	require.True(t, lines["node"], help)
	for _, name := range nodeCommandNames {
		require.False(t, lines[name], "%s must not be listed:\n%s", name, help)
	}
	// Visible at top level, unchanged.
	for _, name := range []string{"version", "status", "config", "keys", "localnet", "tx", "query"} {
		require.True(t, lines[name], "%s must be listed:\n%s", name, help)
	}
}

// legacyInvocations are real forms used by cosmovisor, cosmopilot, Docker
// images, systemd units, localnet scripts, and the upgrade tester. Each must
// resolve and parse its flags as on main, including `--home` before the
// subcommand.
var legacyInvocations = [][]string{
	{"start", "--home", "H", "--trace-store", "F"},
	{"start", "--grpc-only", "--home", "H"},
	{"start", "--home", "H", "--pruning", "nothing", "--log_level", "info"},
	{"init", "M", "--chain-id", "C", "--home", "H"},
	{"init", "M", "--chain-id", "C", "--overwrite"},
	{"init", "none", "--chain-id", "none"},
	{"genesis", "add-genesis-account", "A", "1unibi", "--home", "H"},
	{"genesis", "gentx", "K", "1unibi", "--chain-id", "C", "--keyring-backend", "test", "--yes"},
	{"genesis", "collect-gentxs", "--home", "H"},
	{"genesis", "validate-genesis", "--home", "H"},
	{"genesis", "add-sudo-root-account", "A"},
	{"genesis", "migrate", "v1", "file.json"},
	{"--home", "H", "init", "M"},
	{"--home", "H", "genesis", "gentx", "K", "1unibi"},
	{"--home", "H", "tendermint", "show-node-id"},
	{"--home", "H", "genesis", "collect-gentxs"},
	{"tendermint", "show-node-id"},
	{"tendermint", "show-validator"},
	{"tendermint", "show-address"},
	{"tendermint", "unsafe-reset-all", "--home", "H"},
	{"tendermint", "unsafe_reset_all"},
	{"tendermint", "reset-state"},
	{"tendermint", "reset_state"},
	{"tendermint", "bootstrap-state"},
	{"tendermint", "version"},
	{"export", "--home", "H", "--output-document", "F"},
	{"prune"},
	{"rollback"},
	{
		"testnet", "--v", "4", "--output-dir", "D", "--chain-id", "C",
		"--node-daemon-home", "nibid", "--starting-ip-address", "IP",
		"--keyring-backend", "test",
	},
	{"evm-tx-index", "last-indexed", "latest", "--home", "H"},
}

// parseLegacy resolves args to a command and parses its flags.
func parseLegacy(t *testing.T, rootCmd *cobra.Command, args []string) *cobra.Command {
	t.Helper()
	cmd, _, err := rootCmd.Find(args)
	require.NoError(t, err, args)
	require.NotEqual(t, rootCmd, cmd, "args %v resolved to the root", args)
	_ = cmd.InheritedFlags() // merges root persistent flags like --home
	require.NoError(t, cmd.ParseFlags(flagArgs(rootCmd, args)), args)
	return cmd
}

// flagArgs drops the command-path words so ParseFlags sees only flags and
// positional args, which cobra accepts in any order.
func flagArgs(rootCmd *cobra.Command, args []string) []string {
	_, rest, _ := rootCmd.Find(args)
	return rest
}

// newRootWithGlobalFlags builds the root command and adds the persistent flags
// that svrcmd.Execute adds at run time (--home, --log_level, ...).
func newRootWithGlobalFlags() *cobra.Command {
	rootCmd, _ := nibid.NewRootCmd()
	rootCmd.PersistentFlags().String(flags.FlagLogLevel, "info", "")
	rootCmd.PersistentFlags().String(flags.FlagLogFormat, "plain", "")
	rootCmd.PersistentFlags().Bool(flags.FlagLogNoColor, false, "")
	cmtcli.PrepareBaseCmd(rootCmd, "", app.DefaultNodeHome)
	return rootCmd
}

func TestLegacyNodeInvocations(t *testing.T) {
	rootCmd := newRootWithGlobalFlags()
	for _, args := range legacyInvocations {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := parseLegacy(t, rootCmd, args)
			// The `node` form of the same invocation resolves to the same
			// command name.
			withNode := []string{}
			inserted := false
			for i, a := range args {
				if !inserted && !strings.HasPrefix(a, "-") &&
					(i == 0 || !strings.HasPrefix(args[i-1], "-") || args[i-1] == "--grpc-only") {
					withNode = append(withNode, "node")
					inserted = true
				}
				withNode = append(withNode, a)
			}
			moved := parseLegacy(t, rootCmd, withNode)
			require.Equal(t, cmd.Name(), moved.Name(), withNode)
			require.Equal(t, cmd.CommandPath()[len("nibid "):],
				strings.TrimPrefix(moved.CommandPath(), "nibid node "), withNode)
		})
	}
}

func TestVisibleTopLevelUnchanged(t *testing.T) {
	rootCmd, _ := nibid.NewRootCmd()
	for _, args := range [][]string{
		{"version"}, {"version", "--long", "--output", "json"}, {"status"},
		{"config", "query-mode"}, {"keys", "list"}, {"localnet", "--script", "x"},
	} {
		cmd, _, err := rootCmd.Find(args)
		require.NoError(t, err, args)
		require.False(t, cmd.Hidden, args)
		require.NotEqual(t, rootCmd, cmd, args)
	}
}

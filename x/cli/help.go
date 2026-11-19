package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// RootExample is one copy-pasteable line in the root help.
type RootExample struct {
	Label   string
	Command string
}

// RootExamples are the commands shown in `nibid --help`. A test resolves each
// command path against the real command tree, so a rename breaks the build
// instead of the help text.
var RootExamples = []RootExample{
	{"Check a balance", "nibid q bank balances <address>"},
	{"Send tokens", "nibid tx bank send <from-key> <to-address> 1000000unibi"},
	{"Look up a transaction", "nibid q tx <hash>"},
	{"Call a contract", "nibid tx wasm execute <contract> '<json>' --from <key>"},
	{"Query a contract", "nibid q wasm contract-state smart <contract> '<json>'"},
	{"Decode anything", "nibid debug <tx, payload, address, or pubkey>"},
	{"List keys", "nibid keys list"},
	{"Choose a network", "nibid config node <rpc-url>"},
	{"Start a local chain", "nibid localnet --script | bash"},
}

const RootHelpFooter = `Run "nibid <command> --help" for details, and add --help-verbose to see every flag.`

// renderRootUsage builds the usage text of the root command: examples, the
// command groups, the flags, and a closing hint. It replaces the generic
// "Usage: nibid [command]" block. Subcommands keep cobra's usage template.
func renderRootUsage(root *cobra.Command) string {
	var b strings.Builder

	b.WriteString("Examples:\n")
	labelWidth := 0
	for _, ex := range RootExamples {
		labelWidth = max(labelWidth, len(ex.Label))
	}
	for _, ex := range RootExamples {
		fmt.Fprintf(&b, "  %-*s  %s\n", labelWidth, ex.Label, ex.Command)
	}

	rows := rootRows(root)
	namePad := 0
	for _, c := range rows {
		namePad = max(namePad, len(c.Name()))
	}
	b.WriteString("\nCommands:\n")
	for _, c := range rows {
		desc := c.Short
		if short, ok := rootRowShorts[c.Name()]; ok {
			desc = short
		}
		if len(c.Aliases) > 0 {
			desc += " (alias: " + strings.Join(c.Aliases, ", ") + ")"
		}
		fmt.Fprintf(&b, "  %-*s  %s\n", namePad, c.Name(), desc)
	}

	fmt.Fprintf(&b, "\nFlags:\n%s\n",
		strings.TrimRight(root.LocalFlags().FlagUsages(), " \n"))
	fmt.Fprintf(&b, "\n%s\n", RootHelpFooter)
	return b.String()
}

// setRootUsage swaps the usage of the root command only. Subcommands inherit
// the usage function, so it hands them to the previous one.
func setRootUsage(root *cobra.Command) {
	prev := root.UsageFunc()
	root.SetUsageFunc(func(c *cobra.Command) error {
		if c.HasParent() {
			return prev(c)
		}
		_, err := fmt.Fprint(c.OutOrStderr(), renderRootUsage(c))
		return err
	})
}

// rootRowOrder is the order of the command list in `nibid --help`: daily
// commands first, then node operation, then help and version.
var rootRowOrder = []string{
	"tx", "q", "keys", "config", "status", "debug", "localnet", "node",
	"help", "version",
}

// rootRowShorts say what a person does with each root command.
var rootRowShorts = map[string]string{
	"tx":       "Send transactions",
	"q":        "Query the chain",
	"keys":     "Manage keys",
	"config":   "Set the network and CLI defaults",
	"status":   "Show a node's status",
	"debug":    "Decode and convert anything: txs, payloads, addresses, keys",
	"localnet": "Run a local single-node chain",
	"node":     "Run and maintain a node (start, init, genesis, bft, and more)",
	"help":     "Show help for any command",
	"version":  "Print the binary version",
}

// setRootRowShorts sets the descriptions shown in the root command list.
func setRootRowShorts(root *cobra.Command) {
	for _, c := range root.Commands() {
		if short, ok := rootRowShorts[c.Name()]; ok {
			c.Short = short
		}
	}
}

// rootRows returns the visible root commands in rootRowOrder. Hidden legacy
// commands never appear. Any visible command missing from the order follows
// at the end, so a new command is listed instead of lost.
func rootRows(root *cobra.Command) []*cobra.Command {
	byName := map[string]*cobra.Command{}
	for _, c := range root.Commands() {
		if c.IsAvailableCommand() || c.Name() == "help" {
			byName[c.Name()] = c
		}
	}
	var rows []*cobra.Command
	for _, name := range rootRowOrder {
		if c, ok := byName[name]; ok {
			rows = append(rows, c)
			delete(byName, name)
		}
	}
	rest := []string{}
	for name := range byName {
		rest = append(rest, name)
	}
	sort.Strings(rest)
	for _, name := range rest {
		rows = append(rows, byName[name])
	}
	return rows
}

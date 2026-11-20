package impl

import (
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/rpc"
	authcmd "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/client/cli"

	"github.com/NibiruChain/nibiru/v2/app"
)

// QueryCmd returns the root `nibid query` command with all app module queries.
func QueryCmd() *cobra.Command {
	rootQueryCmd := &cobra.Command{
		Use:                        "q",
		Aliases:                    []string{"query"},
		Short:                      "Query the chain",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	rootQueryCmd.AddCommand(
		authcmd.GetAccountCmd(),
		rpc.ValidatorCommand(),
		rpc.BlockCommand(),
		authcmd.QueryTxsByEventsCmd(),
		QueryTxCmd(),
	)

	app.ModuleBasics.AddQueryCommands(rootQueryCmd)
	rootQueryCmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")
	groupSubcommands(rootQueryCmd, queryHelpGroups)

	return rootQueryCmd
}

// TxCmd returns the root `nibid tx` command with all app module transactions.
func TxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Send transactions",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
	)

	app.ModuleBasics.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")
	groupSubcommands(cmd, txHelpGroups)

	return cmd
}

// helpGroup is a titled section of a command's help and the subcommands in it.
type helpGroup struct {
	ID    string
	Title string
	Names []string
}

// everydayModules are the modules used most, listed first in `tx` and `query`.
var everydayModules = []string{
	"bank", "wasm", "evm", "staking", "gov", "distribution", "tokenfactory",
	"ibc-transfer",
}

var txHelpGroups = []helpGroup{
	{ID: "everyday", Title: "Everyday Modules:", Names: everydayModules},
	{ID: "signing", Title: "Offline Signing:", Names: []string{
		"sign", "sign-batch", "multi-sign", "multisign-batch",
		"validate-signatures", "encode", "decode", "broadcast",
	}},
}

var queryHelpGroups = []helpGroup{
	{ID: "everyday", Title: "Everyday Modules:", Names: everydayModules},
	{ID: "chain", Title: "Chain Data:", Names: []string{
		"account", "block", "tx", "txs", "tendermint-validator-set",
	}},
}

// otherGroupID holds every subcommand not named by a group.
const otherGroupID = "other"

// groupSubcommands organizes the help of parent into the given sections, in
// order, followed by "Other Modules:" for the rest. It only sets help
// metadata: no command moves or changes.
func groupSubcommands(parent *cobra.Command, groups []helpGroup) {
	groupOf := map[string]string{}
	for _, g := range groups {
		parent.AddGroup(&cobra.Group{ID: g.ID, Title: g.Title})
		for _, name := range g.Names {
			groupOf[name] = g.ID
		}
	}
	parent.AddGroup(&cobra.Group{ID: otherGroupID, Title: "Other Modules:"})
	for _, c := range parent.Commands() {
		if id, ok := groupOf[c.Name()]; ok {
			c.GroupID = id
		} else {
			c.GroupID = otherGroupID
		}
	}
}

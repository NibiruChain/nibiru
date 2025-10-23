package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/MakeNowJust/heredoc/v2"

	"github.com/NibiruChain/nibiru/v2/x/sudo"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/spf13/cobra"
)

// GetTxCmd returns a cli command for this module's transactions
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        sudo.ModuleName,
		Short:                      fmt.Sprintf("x/%s transaction subcommands", sudo.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// Add subcommands
	txCmd.AddCommand(
		CmdEditSudoers(),
		CmdEditZeroGasActors(),
		CmdChangeRoot(),
	)

	return txCmd
}

// GetQueryCmd returns a cli command for this module's queries
func GetQueryCmd() *cobra.Command {
	moduleQueryCmd := &cobra.Command{
		Use: sudo.ModuleName,
		Short: fmt.Sprintf(
			"Query commands for the x/%s module", sudo.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// Add subcommands
	cmds := []*cobra.Command{
		CmdQuerySudoers(),
		CmdQueryZeroGasActors(),
	}
	for _, cmd := range cmds {
		moduleQueryCmd.AddCommand(cmd)
	}

	return moduleQueryCmd
}

// CmdEditSudoers is a terminal command corresponding to the EditSudoers
// function of the sdk.Msg handler for x/sudo.
func CmdEditSudoers() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-sudoers [edit-json]",
		Args:  cobra.ExactArgs(1),
		Short: "Edit the x/sudo state (sudoers) by adding or removing contracts",
		Example: heredoc.Docf(`
%s tx sudo edit-sudoers <path/to/edit.json> --from=<key_or_address>`, version.AppName),
		Long: heredoc.Doc(`
Adds or removes contracts from the x/sudo state, giving the 
contracts permissioned access to certain bindings in x/wasm.

The edit.json for 'EditSudoers' is of the form:
{
  "action": "add_contracts",
  "contracts": "..."
}

- Valid action types: "add_contracts", "remove_contracts"	
			`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := new(sudo.MsgEditSudoers)

			// marshals contents into the proto.Message to which 'msg' points.
			contents, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			if err = clientCtx.Codec.UnmarshalJSON(contents, msg); err != nil {
				return err
			}

			// Parse the message sender
			from := clientCtx.GetFromAddress()
			msg.Sender = from.String()

			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdChangeRoot is a terminal command corresponding to the ChangeRoot
func CmdChangeRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change-root [new-root-address]",
		Args:  cobra.ExactArgs(1),
		Short: "Change the root address of the x/sudo state",
		Example: strings.TrimSpace(fmt.Sprintf(`
%s tx sudo change-root <new-root-address> --from=<key_or_address>`,
			version.AppName)),
		Long: strings.TrimSpace(
			`Change the root address of the x/sudo state, giving the
				new address, should be executed by the current root address.
			`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// marshals contents into the proto.Message (msg)
			msg := new(sudo.MsgChangeRoot)
			root := args[0]
			from := clientCtx.GetFromAddress() // Parse the message sender
			msg.Sender = from.String()
			msg.NewRoot = root

			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdEditZeroGasActors is a terminal command that broadcasts a
// "nibiru.sudo.v1.MsgEditZeroGasActors" transaction.
func CmdEditZeroGasActors() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-zero-gas [actors_json_string]",
		Args:  cobra.ExactArgs(1),
		Short: "Change the zero gas actors of the x/sudo state",
		Example: heredoc.Docf(`
%s tx sudo edit-zero-gas [actors_json_string] --from=<key_or_address>

The [actors_json_string] for "ZeroGasActors" is of the form:
{
  "senders": ["nibi1...", "nibi1...", ... ],
  "contracts": ["0x...", "nibi1....", ... ]
}
`, version.AppName),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			jsonBz := []byte(args[0])
			// JSON validation
			rawBz := wasm.RawContractMessage(jsonBz)
			err = rawBz.ValidateBasic()
			if err != nil {
				return fmt.Errorf(`invalid arg "%s" is not a JSON string: %w`, jsonBz, err)
			}

			var zeroGasActors sudo.ZeroGasActors
			err = json.Unmarshal(jsonBz, &zeroGasActors)
			if err != nil {
				return fmt.Errorf("failed to unpack actors json string: %w", err)
			}

			// Parse the message sender
			from := clientCtx.GetFromAddress()

			msg := &sudo.MsgEditZeroGasActors{
				Actors: zeroGasActors,
				Sender: from.String(),
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdQuerySudoers() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "displays the internal state (sudoers) of the x/sudo module",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := sudo.NewQueryClient(clientCtx)

			req := new(sudo.QuerySudoersRequest)
			resp, err := queryClient.QuerySudoers(
				cmd.Context(), req,
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryZeroGasActors() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zero-gas-actors",
		Short: "displays the ZeroGasActors state of the x/sudo module",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := sudo.NewQueryClient(clientCtx)

			req := new(sudo.QueryZeroGasActorsRequest)
			resp, err := queryClient.QueryZeroGasActors(
				cmd.Context(), req,
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

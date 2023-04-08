package cli

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govclientrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

func NewProposalHandler(cliHandler govclient.CLIHandlerFn) govclient.ProposalHandler {
	return govclient.NewProposalHandler(
		/* govclient.CLIHandlerFn */ cliHandler,
		/* govclient.RESTHandlerFn */ func(context client.Context) govclientrest.ProposalRESTHandler {
			return govclientrest.ProposalRESTHandler{
				SubRoute: "deprecated",
				Handler: func(writer http.ResponseWriter, request *http.Request) {
					// The govclient.RESTHandlerFn is entirely removed in sdk v0.46
					_, _ = writer.Write([]byte("deprecated"))
					writer.WriteHeader(http.StatusMethodNotAllowed)
				},
			}
		})
}

var (
	CreatePoolProposalHandler         = NewProposalHandler(CmdCreatePoolProposal)
	EditPoolConfigProposalHandler     = NewProposalHandler(CmdEditPoolConfigProposal)
	EditSwapInvariantsProposalHandler = NewProposalHandler(CmdEditSwapInvariantsProposal)
)

// CmdCreatePoolProposal implements the client command to submit a governance
// proposal to whitelist an oracle for specified asset pairs.
func CmdCreatePoolProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pool [proposal-json] --deposit=[deposit]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to create a new market",
		Example: strings.TrimSpace(fmt.Sprintf(`
			Example: 
			$ %s tx gov submit-proposal create-pool <path/to/proposal.json> --deposit="1000unibi" --from=<key_or_address> 
			`, version.AppName)),
		Long: strings.TrimSpace(
			`Submits a proposal to create a new market, which in turn create a new x/perp market

			A proposal.json for 'CreatePoolProposal' contains:
			{
			  "title": "Create market for ETH:USDT",
			  "description": "I wanna get liquidated on ETH:USDT",
			  "pair": "ETH:USDT",
			  "trade_limit_ratio": "0.2",
              ...
			}
			`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			proposal := &types.CreatePoolProposal{}
			contents, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			// marshals the contents into the proto.Message to which 'proposal' points.
			if err = clientCtx.Codec.UnmarshalJSON(contents, proposal); err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(proposal, deposit, from)
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(
		/*name=*/ govcli.FlagDeposit,
		/*defaultValue=*/ "",
		/*usage=*/ "governance deposit for proposal")
	if err := cmd.MarkFlagRequired(govcli.FlagDeposit); err != nil {
		panic(err)
	}

	return cmd
}

// CmdEditPoolConfigProposal implements the client command to submit a governance
// proposal to whitelist an oracle for specified asset pairs.
func CmdEditPoolConfigProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-pool-cfg [proposal-json] --deposit=[deposit]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to edit the market config",
		Example: strings.TrimSpace(fmt.Sprintf(`
			Example: 
			$ %s tx gov submit-proposal edit-pool-cfg <path/to/proposal.json> --deposit="1000unibi" --from=<key_or_address> 
			`, version.AppName)),
		Long: strings.TrimSpace(
			`Submits a proposal to edit a market's config, it's parameters that 
			aren't based on the reserves (e.g. max leverage, maintenance margin ratio).

			A proposal.json for 'EditPoolConfigProposal' contains:
			{
			  "title": "Edit market config for NIBI:NUSD",
			  "description": "I want to take 100x leverage on my NIBI",
			  "pair": "unibi:unusd",
			  "config": {
				"max_leverage": "100",
				"trade_limit_ratio": "0.1",
				"fluctuation_limit_ratio": "0.1",
				"max_oracle_spread_ratio": "0.1",
				"maintenance_margin_ratio": "0.01"
			  }
			}
			`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			proposal := &types.EditPoolConfigProposal{}
			contents, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			// marshals the contents into the proto.Message to which 'proposal' points.
			if err = clientCtx.Codec.UnmarshalJSON(contents, proposal); err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(proposal, deposit, from)
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(
		/*name=*/ govcli.FlagDeposit,
		/*defaultValue=*/ "",
		/*usage=*/ "governance deposit for proposal")
	if err := cmd.MarkFlagRequired(govcli.FlagDeposit); err != nil {
		panic(err)
	}

	return cmd
}

// CmdEditSwapInvariantsProposal implements the client command to submit a governance
// proposal to whitelist an oracle for specified asset pairs.
func CmdEditSwapInvariantsProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-invariant [proposal-json] --deposit=[deposit]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to edit the market config",
		Example: strings.TrimSpace(fmt.Sprintf(`
			Example: 
			$ %s tx gov submit-proposal edit-invariant <path/to/proposal.json> --deposit="1000unibi" --from=<key_or_address> 
			`, version.AppName)),
		Long: strings.TrimSpace(
			`Submits a proposal to edit the swap invariant of one or more markets.

			A proposal.json for 'EditSwapInvariantsProposal' contains:
			{
		      "title": "NIP-4: Change the swap invariant for ATOM, OSMO, and BTC.",
			  "description": "increase swap invariant for many virtual pools",
			  "swap_invariant_maps": [
			    {"pair": "uatom:unusd", "multiplier": "2"},
			    {"pair": "uosmo:unusd", "multiplier": "5"},
			    {"pair": "ubtc:unusd", "multiplier": "100"}
			  ]
			}
			`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			proposal := &types.EditSwapInvariantsProposal{}
			contents, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			// marshals the contents into the proto.Message to which 'proposal' points.
			if err = clientCtx.Codec.UnmarshalJSON(contents, proposal); err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(proposal, deposit, from)
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(
		/*name=*/ govcli.FlagDeposit,
		/*defaultValue=*/ "",
		/*usage=*/ "governance deposit for proposal")
	if err := cmd.MarkFlagRequired(govcli.FlagDeposit); err != nil {
		panic(err)
	}

	return cmd
}

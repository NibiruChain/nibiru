package cli

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"

	"github.com/spf13/cobra"
)

var (
	AddOracleProposalHandler = govclient.NewProposalHandler(
		/* govclient.CLIHandlerFn */ CmdAddOracleProposal,
		/* govclient.RESTHandlerFn */ AddOracleProposalRESTHandler)
)

// CmdAddOracleProposal implements the client command to submit a governance
// proposal to whitelist an oracle for specified asset pairs.
func CmdAddOracleProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-oracle [proposal-json]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to whitelist an oracle",
		Example: strings.TrimSpace(fmt.Sprintf(`
			Example: 
			$ %s tx gov add-oracle <path/to/proposal.json> --from=<key_or_address>
			`, version.AppName)),
		Long: strings.TrimSpace(
			`Submits a proposal to whitelist an oracle on specified pairs

			A proposal.json for 'AddOracleProposal' contains:
			{
			  "title": "Cataclysm-004",
			  "description": "Whitelists Delphi to post prices for OHM",
			  "oracle": "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
			  "pairs": ["uohm:uusd"],
			  "deposit": "1000unibi"
			}
			`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			proposal := &types.AddOracleProposalWithDeposit{}
			contents, err := ioutil.ReadFile(args[0])
			if err != nil {
				return err
			}

			// marshals the contents into the proto.Message to which 'proposal' points.
			if err = clientCtx.Codec.UnmarshalJSON(contents, proposal); err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			content := types.NewAddOracleProposal(
				proposal.Title,
				proposal.Description,
				proposal.Oracle,
				proposal.Pairs,
			)
			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

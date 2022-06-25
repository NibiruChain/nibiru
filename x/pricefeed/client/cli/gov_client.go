package cli

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
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
		Use:   "add-oracle [proposal-json] --deposit=[deposit]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to whitelist an oracle",
		Example: strings.TrimSpace(fmt.Sprintf(`
			Example: 
			$ %s tx gov submit-proposal add-oracle <path/to/proposal.json> --deposit="1000unibi" --from=<key_or_address> 
			`, version.AppName)),
		Long: strings.TrimSpace(
			`Submits a proposal to whitelist an oracle on specified pairs

			A proposal.json for 'AddOracleProposal' contains:
			{
			  "title": "Cataclysm-004",
			  "description": "Whitelists Delphi to post prices for OHM",
			  "oracles": ["nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"],
			  "pairs": ["uohm:uusd"]
			}
			`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			proposal := &types.AddOracleProposal{}
			contents, err := ioutil.ReadFile(args[0])
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

			content := types.NewAddOracleProposal(
				proposal.Title,
				proposal.Description,
				proposal.Oracles,
				proposal.Pairs,
			)
			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
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

package cli

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"ngithub.com/cosmos/cosmos-sdk/client"

	// "github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

var (
	AddProposalHandler = govclient.NewProposalHandler(CmdAddOracleProposal, RESTHandlers.AddOracleProposalHandler)
)

type RESTHandlers struct {
	AddOracleProposalHandler govclient.RESTHandlerFn
	// RemoveOracleProposalHandler
}

// CmdAddOracleProposal implements the client command to submit a governance
// proposal to whitelist an oracle for specified asset pairs.
func CmdAddOracleProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-oracle [proposal-json]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to whitelist an oracle",
		Long: strings.TrimSpace(fmt.Sprintf(
			`TODO docs`,
			version.AppName,
		)),
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

			// marshals the contents into the the to which 'proposal' points.
			if err = clientCtx.Codec.UnmarshalJSON(contents, proposal); err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			proposalMsg := types.NewAddOracleProposal(
				proposal.Title,
				proposal.Description,
				proposal.Oracle,
				proposal.Pairs,
			)
			msg, err := govtypes.NewMsgSubmitProposal(proposalMsg, deposit, from)
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)

		},
	}

	return cmd
}

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

	"github.com/NibiruChain/nibiru/x/vpool/types"
)

var (
	CreatePoolProposalHandler = govclient.NewProposalHandler(
		/* govclient.CLIHandlerFn */ CmdCreatePoolProposal,
		/* govclient.RESTHandlerFn */ func(context client.Context) govclientrest.ProposalRESTHandler {
			return govclientrest.ProposalRESTHandler{
				SubRoute: "create_pool",
				Handler: func(writer http.ResponseWriter, request *http.Request) {
					_, _ = writer.Write([]byte("deprecated"))
					writer.WriteHeader(http.StatusMethodNotAllowed)
				},
			}
		})
)

// CmdCreatePoolProposal implements the client command to submit a governance
// proposal to whitelist an oracle for specified asset pairs.
func CmdCreatePoolProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pool [proposal-json] --deposit=[deposit]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to create a new vpool",
		Example: strings.TrimSpace(fmt.Sprintf(`
			Example: 
			$ %s tx gov submit-proposal create-pool <path/to/proposal.json> --deposit="1000unibi" --from=<key_or_address> 
			`, version.AppName)),
		Long: strings.TrimSpace(
			`Submits a proposal to create a new vpool, which in turn create a new x/perp market

			A proposal.json for 'CreatePoolProposal' contains:
			{
			  "title": "Create vpool for ETH:USDT",
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

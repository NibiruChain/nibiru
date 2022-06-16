package cli

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/version"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govclientrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"

	"github.com/spf13/cobra"
)

var (
	AddOracleProposalHandler = govclient.NewProposalHandler(
		/* govclient.CLIHandlerFn */ CmdAddOracleProposal,
		/* govclient.RESTHandlerFn */ AddOracleProposalRESTHandler)
)

/* AddOracleProposalRESTHandler defines a REST handler an 'AddOracleProposal'.
The sub-route is mounted on the governance REST handler.
*/
func AddOracleProposalRESTHandler(clientCtx client.Context) govclientrest.ProposalRESTHandler {
	/* restHandlerFnAddOracleProposal is an HTTP handler for an 'AddOracleProposal'.
	A 'HandlerFunc' type is an adapter to allow the use of ordinary functions as HTTP
	handlers. If f is a function with the appropriate signature, HandlerFunc(f)
	is a Handler that calls f.
	*/
	restHandlerFnAddOracleProposal := func(clientCtx client.Context) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var req AddOracleProposalHttpRequest
			if !rest.ReadRESTReq(w, r, clientCtx.LegacyAmino, &req) {
				return
			}

			req.BaseReq = req.BaseReq.Sanitize()
			if !req.BaseReq.ValidateBasic(w) {
				return
			}

			content := types.NewAddOracleProposal(
				req.Title,
				req.Description,
				req.Oracle.String(),
				req.Pairs,
			)
			msg, err := govtypes.NewMsgSubmitProposal(content, req.Deposit, req.Proposer)
			if rest.CheckBadRequestError(w, err) {
				return
			}
			if rest.CheckBadRequestError(w, msg.ValidateBasic()) {
				return
			}

			tx.WriteGeneratedTxResponse(clientCtx, w, req.BaseReq, msg)
		}
	}

	return govclientrest.ProposalRESTHandler{
		SubRoute: "add_oracle",
		Handler:  restHandlerFnAddOracleProposal(clientCtx),
	}
}

type (
	AddOracleProposalHttpRequest struct {
		BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`

		Proposer    sdk.AccAddress `json:"proposer" yaml:"proposer"`
		Title       string         `json:"title" yaml:"title"`
		Description string         `json:"description" yaml:"description"`
		Oracle      sdk.AccAddress `json:"oracle" yaml:"oracle"`
		Pairs       []string       `json:"pairs" yaml:"pairs"`
		Deposit     sdk.Coins      `json:"deposit" yaml:"deposit"`
	}
)

// CmdAddOracleProposal implements the client command to submit a governance
// proposal to whitelist an oracle for specified asset pairs.
func CmdAddOracleProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-oracle [proposal-json]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to whitelist an oracle",
		Long: strings.TrimSpace(fmt.Sprintf(
			`TODO docs
			Example: 
			$ %s tx gov add-oracle <path/to/proposal.json> --from=<key_or_address>
			
			A proposal.json for 'AddOracleProposal' contains:
			{
			  "title": "Cataclysm-004",
			  "description": "Whitelists Delphi to post prices for OHM",
			  "oracle": "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
			  "pairs": ["uohm:uusd"],
			  "deposit": "1000unibi"
			}
			`,
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

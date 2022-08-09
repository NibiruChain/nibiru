package cli

import (
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	govclientrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

/*
	AddOracleProposalRESTHandler defines a REST handler an 'AddOracleProposal'.

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
				req.Oracles,
				req.Pairs,
			)

			fromAddr, err := sdk.AccAddressFromBech32(req.BaseReq.From)
			if rest.CheckBadRequestError(w, err) {
				return
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, req.Deposit, fromAddr)
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

		Title       string    `json:"title" yaml:"title"`
		Description string    `json:"description" yaml:"description"`
		Oracles     []string  `json:"oracle" yaml:"oracles"`
		Pairs       []string  `json:"pairs" yaml:"pairs"`
		Deposit     sdk.Coins `json:"deposit" yaml:"deposit"`
	}
)

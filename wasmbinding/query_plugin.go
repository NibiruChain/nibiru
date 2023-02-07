package wasmbinding

import (
	"encoding/json"

	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	// perp "github.com/NibiruChain/nibiru/x/perp"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CustomQuerier dispatches custom CosmWasm bindings queries.
func CustomQuerier(qp *QueryPlugin) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var contractQuery bindings.NibiruQuery
		if err := json.Unmarshal(request, &contractQuery); err != nil {
			return nil, sdkerrors.Wrap(err, "nibiru query")
		}

		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown Custom variant"}
	}
}

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

		switch {
		case contractQuery.Position != nil:
			res, err := qp.GetPosition(ctx, contractQuery.Position.Trader, contractQuery.Position.Pair)
			if err != nil {
				return nil, sdkerrors.Wrap(err, "position")
			}

			bz, err := json.Marshal(res)
			if err != nil {
				return nil, sdkerrors.Wrap(err, "position")
			}

			return bz, nil
		case contractQuery.Positions != nil:
			res, err := qp.GetPositions(ctx, contractQuery.Position.Trader)
			if err != nil {
				return nil, sdkerrors.Wrap(err, "positions")
			}

			bz, err := json.Marshal(res)
			if err != nil {
				return nil, sdkerrors.Wrap(err, "positions")
			}

			return bz, nil
		default:
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown Custom variant"}
		}
	}
}

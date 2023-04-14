package binding

import (
	"encoding/json"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CustomQuerier returns a function that is an implementation of custom querier mechanism for specific messages
func CustomQuerier(qp *QueryPlugin) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var wasmContractQuery cw_struct.BindingQuery
		if err := json.Unmarshal(request, &wasmContractQuery); err != nil {
			return nil, sdkerrors.Wrapf(err, "failed to JSON unmarshal nibiru query: %v", err)
		}

		switch {
		case wasmContractQuery.AllMarkets != nil:
			cwResp, err := qp.Perp.AllMarkets(ctx)
			if err != nil {
				return nil, sdkerrors.Wrapf(err,
					"failed to query: perp all markets: request: %v",
					wasmContractQuery.AllMarkets)
			}
			bz, err := json.Marshal(cwResp)
			if err != nil {
				return nil, sdkerrors.Wrapf(err, "failed to JSON marshal response: %v", cwResp)
			}
			return bz, nil
		case wasmContractQuery.Reserves != nil:
			cwResp, err := qp.Perp.Reserves(ctx, wasmContractQuery.Reserves)
			if err != nil {
				return nil, sdkerrors.Wrapf(err,
					"failed to query: perp reserves: request: %v",
					wasmContractQuery.AllMarkets)
			}
			bz, err := json.Marshal(cwResp)
			if err != nil {
				return nil, sdkerrors.Wrapf(err, "failed to JSON marshal response: %v", cwResp)
			}
			return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.BasePrice != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.Positions != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.Position != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.PremiumFraction != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.Metrics != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.ModuleAccounts != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.PerpParams != nil:
		// 	return bz, nil
		default:
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown nibiru query variant"}
		}
	}

}

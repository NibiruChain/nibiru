package wasmbinding

import (
	"encoding/json"

	sdkerrors "cosmossdk.io/errors"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

type QueryPlugin struct {
	Oracle *OracleQuerier
}

// NewQueryPlugin returns a pointer to a new QueryPlugin
func NewQueryPlugin(oracle oraclekeeper.Keeper) QueryPlugin {
	return QueryPlugin{
		Oracle: &OracleQuerier{
			oracle: oraclekeeper.NewQuerier(oracle),
		},
	}
}

func (qp *QueryPlugin) ToBinary(
	cwResp any, err error, cwReq any,
) ([]byte, error) {
	if err != nil {
		return nil, sdkerrors.Wrapf(err,
			"failed to query: perp all markets: request: %v",
			cwReq)
	}
	bz, err := json.Marshal(cwResp)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to JSON marshal response: %v", cwResp)
	}
	return bz, nil
}

// CustomQuerier returns a function that is an implementation of the custom
// querier mechanism for specific messages
func CustomQuerier(qp QueryPlugin) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var wasmContractQuery bindings.BindingQuery
		if err := json.Unmarshal(request, &wasmContractQuery); err != nil {
			return nil, sdkerrors.Wrapf(err, "failed to JSON unmarshal nibiru query: %v", err)
		}

		switch {
		// Add additional query types here
		case wasmContractQuery.OraclePrices != nil:
			cwReq := wasmContractQuery.OraclePrices
			cwResp, err := qp.Oracle.ExchangeRates(ctx, cwReq)
			return qp.ToBinary(cwResp, err, cwReq)

		default:
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown nibiru query variant"}
		}
	}
}

// ----------------------------------------------------------------------
// OracleQuerier
// ----------------------------------------------------------------------

type OracleQuerier struct {
	oracle oracletypes.QueryServer
}

func (oracleExt *OracleQuerier) ExchangeRates(
	ctx sdk.Context, cwReq *bindings.OraclePrices,
) (*bindings.OraclePricesResponse, error) {
	queryExchangeRatesRequest := oracletypes.QueryExchangeRatesRequest{}
	queryExchangeRates, err := oracleExt.oracle.ExchangeRates(ctx, &queryExchangeRatesRequest)

	// Transform Tuple to Map
	exchangeRates := make(map[string]sdk.Dec)
	for _, exchangeRate := range queryExchangeRates.ExchangeRates {
		exchangeRates[exchangeRate.Pair.String()] = exchangeRate.ExchangeRate
	}

	cwResp := new(bindings.OraclePricesResponse)
	*cwResp = exchangeRates
	return cwResp, err
}

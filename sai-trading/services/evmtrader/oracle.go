package evmtrader

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	ethereum "github.com/ethereum/go-ethereum"
)

// queryOraclePrice queries the oracle contract for the current price of a token
func (t *EVMTrader) queryOraclePrice(ctx context.Context, tokenIndex uint64) (float64, error) {
	fmt.Println("queryOraclePrice", tokenIndex)
	// Build query message - oracle expects "index" not "token_id"
	queryMsg := map[string]interface{}{
		"get_price": map[string]interface{}{
			"index": tokenIndex,
		},
	}
	queryBytes, err := json.Marshal(queryMsg)
	if err != nil {
		return 0, fmt.Errorf("marshal query: %w", err)
	}

	// Pack query call
	wasmABI := getWasmPrecompileABI()
	wasmPrecompileAddr := precompile.PrecompileAddr_Wasm
	data, err := wasmABI.Pack("query", t.addrs.OracleAddress, queryBytes)
	if err != nil {
		return 0, fmt.Errorf("pack query: %w", err)
	}

	// Call contract (read-only, no transaction needed)
	msg := ethereum.CallMsg{
		From: t.accountAddr,
		To:   &wasmPrecompileAddr,
		Data: data,
	}
	result, err := t.client.CallContract(ctx, msg, nil)
	if err != nil {
		return 0, fmt.Errorf("call contract: %w", err)
	}

	// The precompile returns bytes, need to unpack first
	unpacked, err := wasmABI.Unpack("query", result)
	if err != nil {
		return 0, fmt.Errorf("unpack query result: %w", err)
	}
	if len(unpacked) == 0 {
		return 0, fmt.Errorf("empty query result")
	}

	// The result is bytes containing JSON
	responseBytes, ok := unpacked[0].([]byte)
	if !ok {
		return 0, fmt.Errorf("invalid query result type")
	}

	// Parse JSON response
	// The oracle returns PriceResponse { price: Decimal, last_oracle_address: Option<Addr> }
	// Decimal is serialized as a string in JSON
	var priceResponse struct {
		Price string `json:"price"`
	}
	if err := json.Unmarshal(responseBytes, &priceResponse); err != nil {
		// Try with data wrapper (CosmWasm sometimes wraps responses)
		var wrapped struct {
			Data struct {
				Price string `json:"price"`
			} `json:"data"`
		}
		if err2 := json.Unmarshal(responseBytes, &wrapped); err2 != nil {
			return 0, fmt.Errorf("unmarshal query result (direct): %w, (wrapped): %w, raw: %s", err, err2, string(responseBytes))
		}
		if wrapped.Data.Price == "" {
			return 0, fmt.Errorf("price field is empty in wrapped response, raw: %s", string(responseBytes))
		}
		price, err := strconv.ParseFloat(wrapped.Data.Price, 64)
		fmt.Println("price", price)
		if err != nil {
			return 0, fmt.Errorf("parse wrapped price '%s': %w", wrapped.Data.Price, err)
		}
		return price, nil
	}

	if priceResponse.Price == "" {
		return 0, fmt.Errorf("price field is empty in response, raw: %s", string(responseBytes))
	}

	price, err := strconv.ParseFloat(priceResponse.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("parse price '%s': %w", priceResponse.Price, err)
	}

	return price, nil
}

package evmtrader

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// queryWasmContract is a helper method that encapsulates the common pattern
// for querying WASM contracts via the EVM precompile.
func (t *EVMTrader) queryWasmContract(ctx context.Context, contractAddr string, queryMsg map[string]interface{}) ([]byte, error) {
	// Marshal query message to JSON
	queryBytes, err := json.Marshal(queryMsg)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	// Get WASM precompile ABI and address
	wasmABI := getWasmPrecompileABI()
	wasmPrecompileAddr := precompile.PrecompileAddr_Wasm

	// Pack the query call with contract address and query bytes
	data, err := wasmABI.Pack("query", contractAddr, queryBytes)
	if err != nil {
		return nil, fmt.Errorf("pack query: %w", err)
	}

	// Create call message for read-only contract call
	msg := ethereum.CallMsg{
		From: t.accountAddr,
		To:   &wasmPrecompileAddr,
		Data: data,
	}

	// Execute the contract call
	result, err := t.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("call contract: %w", err)
	}

	// Unpack the result using the WASM ABI
	unpacked, err := wasmABI.Unpack("query", result)
	if err != nil {
		return nil, fmt.Errorf("unpack query result: %w", err)
	}
	if len(unpacked) == 0 {
		return nil, fmt.Errorf("empty query result")
	}

	// Extract the bytes containing the JSON response
	responseBytes, ok := unpacked[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid query result type")
	}

	return responseBytes, nil
}

// QueryAndDisplayPositions queries all trades and displays open positions
func (t *EVMTrader) QueryAndDisplayPositions(ctx context.Context) error {
	// Query trades/positions
	trades, err := t.QueryTrades(ctx)
	if err != nil {
		return fmt.Errorf("query trades: %w", err)
	}

	// Filter and display open positions
	openPositions := make([]ParsedTrade, 0)
	for _, trade := range trades {
		if trade.IsOpen {
			openPositions = append(openPositions, trade)
		}
	}

	if len(openPositions) == 0 {
		fmt.Println("No positions found")
		return nil
	}

	fmt.Println("Open Positions:")
	fmt.Println("===============")
	for _, trade := range openPositions {
		fmt.Printf("Trade Index: %d\n", trade.UserTradeIndex)
		fmt.Printf("  Market: %d\n", trade.MarketIndex)
		fmt.Printf("  Type: %s\n", trade.TradeType)
		fmt.Printf("  Direction: %s\n", map[bool]string{true: "LONG", false: "SHORT"}[trade.Long])
		fmt.Printf("  Leverage: %sx\n", trade.Leverage)
		fmt.Printf("  Collateral: %s (index: %d)\n", trade.CollateralAmount, trade.CollateralIndex)
		fmt.Printf("  Open Price: %s\n", trade.OpenPrice)
		if trade.TP != "0" && trade.TP != "" {
			fmt.Printf("  Take Profit: %s\n", trade.TP)
		}
		if trade.SL != nil && *trade.SL != "" && *trade.SL != "null" {
			fmt.Printf("  Stop Loss: %s\n", *trade.SL)
		}
		fmt.Println()
	}

	return nil
}

// QueryMarkets queries the perp contract for all available markets
func (t *EVMTrader) QueryMarkets(ctx context.Context) ([]MarketInfo, error) {
	// First, get the list of market indices
	queryMsg := map[string]interface{}{
		"list_markets": map[string]interface{}{},
	}

	// Execute the query using the helper method
	responseBytes, err := t.queryWasmContract(ctx, t.addrs.PerpAddress, queryMsg)
	if err != nil {
		return nil, err
	}

	// Parse JSON response - list_markets returns an array of market index strings
	var marketIndices []string
	if err := json.Unmarshal(responseBytes, &marketIndices); err != nil {
		// Try with data wrapper
		var wrapped struct {
			Data []string `json:"data"`
		}
		if err2 := json.Unmarshal(responseBytes, &wrapped); err2 != nil {
			return nil, fmt.Errorf("unmarshal query result (direct): %w, (wrapped): %w, raw: %s", err, err2, string(responseBytes))
		}
		marketIndices = wrapped.Data
	}

	// Now query each market individually to get full details
	var markets []MarketInfo
	for _, marketIndexStr := range marketIndices {
		// Extract market index from string (e.g., "MarketIndex(0)")
		var marketIndex uint64
		if _, err := fmt.Sscanf(marketIndexStr, "MarketIndex(%d)", &marketIndex); err != nil {
			// Try parsing as just a number
			if _, err := fmt.Sscanf(marketIndexStr, "%d", &marketIndex); err != nil {
				continue // Skip invalid indices
			}
		}

		// Query individual market details
		market, err := t.queryMarket(ctx, marketIndex)
		if err != nil {
			// Log error but continue with other markets
			t.log("Failed to query market", "market_index", marketIndex, "error", err.Error())
			// Still add the market with just the index
			markets = append(markets, MarketInfo{Index: marketIndex})
			continue
		}
		markets = append(markets, *market)
	}

	return markets, nil
}

// QueryTrades queries the perp contract for all trades/positions for the current user
func (t *EVMTrader) QueryTrades(ctx context.Context) ([]ParsedTrade, error) {
	// Convert Ethereum address to Nibiru Bech32 address
	nibiruAddr := eth.EthAddrToNibiruAddr(t.accountAddr)
	fmt.Println("nibiruAddr", nibiruAddr.String())
	fmt.Println("t.accountAddr", t.accountAddr)

	queryMsg := map[string]interface{}{
		"get_trades": map[string]interface{}{
			"trader": nibiruAddr.String(),
		},
	}

	// Execute the query using the helper method
	responseBytes, err := t.queryWasmContract(ctx, t.addrs.PerpAddress, queryMsg)
	if err != nil {
		return nil, err
	}

	// Parse JSON response - returns an array of trades with wrapped index types
	var rawTrades []Trade
	if err := json.Unmarshal(responseBytes, &rawTrades); err != nil {
		return nil, fmt.Errorf("unmarshal trades response: %w, raw: %s", err, string(responseBytes))
	}

	// Parse wrapped indices to numeric values
	parsedTrades := make([]ParsedTrade, 0, len(rawTrades))
	for _, raw := range rawTrades {
		parsed := ParsedTrade{
			User:              raw.User,
			Leverage:          raw.Leverage,
			Long:              raw.Long,
			IsOpen:            raw.IsOpen,
			TradeType:         raw.TradeType,
			CollateralAmount:  raw.CollateralAmount,
			OpenPrice:         raw.OpenPrice,
			OpenCollateralAmt: raw.OpenCollateralAmt,
			TP:                raw.TP,
			SL:                raw.SL,
			IsEvmOrigin:       raw.IsEvmOrigin,
		}

		// Parse market_index
		var marketIdx uint64
		if _, err := fmt.Sscanf(raw.MarketIndex, "MarketIndex(%d)", &marketIdx); err != nil {
			return nil, fmt.Errorf("parse market_index '%s': expected MarketIndex(N), got: %w", raw.MarketIndex, err)
		}
		parsed.MarketIndex = marketIdx

		// Parse user_trade_index
		var tradeIdx uint64
		if _, err := fmt.Sscanf(raw.UserTradeIndex, "UserTradeIndex(%d)", &tradeIdx); err != nil {
			return nil, fmt.Errorf("parse user_trade_index '%s': expected UserTradeIndex(N), got: %w", raw.UserTradeIndex, err)
		}
		parsed.UserTradeIndex = tradeIdx

		// Parse collateral_index
		var collateralIdx uint64
		if _, err := fmt.Sscanf(raw.CollateralIndex, "TokenIndex(%d)", &collateralIdx); err != nil {
			return nil, fmt.Errorf("parse collateral_index '%s': expected TokenIndex(N), got: %w", raw.CollateralIndex, err)
		}
		parsed.CollateralIndex = collateralIdx

		parsedTrades = append(parsedTrades, parsed)
	}

	return parsedTrades, nil
}

// queryMarket queries a single market by index
func (t *EVMTrader) queryMarket(ctx context.Context, marketIndex uint64) (*MarketInfo, error) {
	queryMsg := map[string]interface{}{
		"get_market": map[string]interface{}{
			"index": fmt.Sprintf("MarketIndex(%d)", marketIndex),
		},
	}

	// Execute the query using the helper method
	responseBytes, err := t.queryWasmContract(ctx, t.addrs.PerpAddress, queryMsg)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var marketData map[string]interface{}
	if err := json.Unmarshal(responseBytes, &marketData); err != nil {
		return nil, fmt.Errorf("unmarshal market query result: %w, raw: %s", err, string(responseBytes))
	}

	market := MarketInfo{Index: marketIndex}

	// The get_market query returns market_info fields directly at root level
	// Check if market_info exists as nested object, otherwise use root level
	var marketInfo map[string]interface{}
	if nested, ok := marketData["market_info"].(map[string]interface{}); ok {
		marketInfo = nested
	} else {
		// Market info fields are at root level
		marketInfo = marketData
	}

	// Parse base token - required field
	base, ok := marketInfo["base"].(string)
	if !ok {
		return nil, fmt.Errorf("base token missing or invalid in market %d, raw: %s", marketIndex, string(responseBytes))
	}
	var baseIdx uint64
	if _, err := fmt.Sscanf(base, "TokenIndex(%d)", &baseIdx); err != nil {
		return nil, fmt.Errorf("parse base token '%s' in market %d: expected TokenIndex(N), got: %w", base, marketIndex, err)
	}
	market.BaseToken = &baseIdx

	// Parse quote token - required field
	quote, ok := marketInfo["quote"].(string)
	if !ok {
		return nil, fmt.Errorf("quote token missing or invalid in market %d, raw: %s", marketIndex, string(responseBytes))
	}
	var quoteIdx uint64
	if _, err := fmt.Sscanf(quote, "TokenIndex(%d)", &quoteIdx); err != nil {
		return nil, fmt.Errorf("parse quote token '%s' in market %d: expected TokenIndex(N), got: %w", quote, marketIndex, err)
	}
	market.QuoteToken = &quoteIdx

	// Parse max_oi if present
	if maxOI, ok := marketData["max_oi"].(string); ok {
		market.MaxOI = &maxOI
	}

	// Parse fee_per_block if present
	if feePerBlock, ok := marketData["fee_per_block"].(string); ok {
		market.FeePerBlock = &feePerBlock
	}

	return &market, nil
}

// queryOraclePrice queries the oracle contract for the current price of a token
func (t *EVMTrader) queryOraclePrice(ctx context.Context, tokenIndex uint64) (float64, error) {
	t.log("Querying oracle price", "token_index", tokenIndex)
	// Build query message - oracle expects "index" not "token_id"
	queryMsg := map[string]interface{}{
		"get_price": map[string]interface{}{
			"index": tokenIndex,
		},
	}

	// Execute the query using the helper method
	responseBytes, err := t.queryWasmContract(ctx, t.addrs.OracleAddress, queryMsg)
	if err != nil {
		return 0, err
	}

	// Parse JSON response
	// The oracle returns PriceResponse { price: Decimal, last_oracle_address: Option<Addr> }
	// Decimal is serialized as a string in JSON
	var priceResponse struct {
		Price string `json:"price"`
	}
	if err := json.Unmarshal(responseBytes, &priceResponse); err != nil {
		return 0, fmt.Errorf("unmarshal price response: %w, raw: %s", err, string(responseBytes))
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

// queryExchangeRate queries the oracle contract for the exchange rate between base and quote tokens
// This matches what the perp contract does - it queries GetExchangeRate to get base_per_quote
func (t *EVMTrader) queryExchangeRate(ctx context.Context, baseIndex, quoteIndex uint64) (float64, error) {
	t.log("Querying oracle exchange rate", "base_index", baseIndex, "quote_index", quoteIndex)
	// Build query message - oracle expects GetExchangeRate with base and quote
	queryMsg := map[string]interface{}{
		"get_exchange_rate": map[string]interface{}{
			"base":  baseIndex,
			"quote": quoteIndex,
		},
	}

	// Execute the query using the helper method
	responseBytes, err := t.queryWasmContract(ctx, t.addrs.OracleAddress, queryMsg)
	if err != nil {
		return 0, err
	}

	// Parse JSON response
	// The oracle returns ExchangeRateResp { base_per_quote: Decimal, ... }
	var exchangeRateResp struct {
		BasePerQuote string `json:"base_per_quote"`
	}
	if err := json.Unmarshal(responseBytes, &exchangeRateResp); err != nil {
		return 0, fmt.Errorf("unmarshal exchange rate response: %w, raw: %s", err, string(responseBytes))
	}

	if exchangeRateResp.BasePerQuote == "" {
		return 0, fmt.Errorf("base_per_quote field is empty in response, raw: %s", string(responseBytes))
	}

	rate, err := strconv.ParseFloat(exchangeRateResp.BasePerQuote, 64)
	if err != nil {
		return 0, fmt.Errorf("parse base_per_quote '%s': %w", exchangeRateResp.BasePerQuote, err)
	}

	return rate, nil
}

// queryERC20Balance queries the ERC20 balance of an account
func (t *EVMTrader) queryERC20Balance(ctx context.Context, erc20ABI abi.ABI, token common.Address, account common.Address) (*big.Int, error) {
	data, err := erc20ABI.Pack("balanceOf", account)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{
		From: account,
		To:   &token,
		Data: data,
	}
	out, err := t.client.CallContract(ctx, msg, nil)
	if err != nil {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(out), nil
}

// QueryCollaterals queries the perp contract for all available collaterals
// Tries to list collaterals, and if that's not supported, tries common indices (0-10)
func (t *EVMTrader) QueryCollaterals(ctx context.Context) ([]CollateralInfo, error) {
	// Try list_collaterals query (similar to list_markets)
	queryMsg := map[string]interface{}{
		"list_collaterals": map[string]interface{}{},
	}

	responseBytes, err := t.queryWasmContract(ctx, t.addrs.PerpAddress, queryMsg)
	if err == nil {
		// Parse JSON response - list_collaterals might return an array of collateral indices
		var collateralIndices []string
		if err := json.Unmarshal(responseBytes, &collateralIndices); err != nil {
			// Try with data wrapper
			var wrapped struct {
				Data []string `json:"data"`
			}
			if err2 := json.Unmarshal(responseBytes, &wrapped); err2 == nil {
				collateralIndices = wrapped.Data
			}
		}

		if len(collateralIndices) > 0 {
			// Query each collateral individually to get full details
			var collaterals []CollateralInfo
			for _, collateralIndexStr := range collateralIndices {
				// Extract collateral index from string (e.g., "TokenIndex(0)" or just "0")
				var collateralIndex uint64
				if _, err := fmt.Sscanf(collateralIndexStr, "TokenIndex(%d)", &collateralIndex); err != nil {
					// Try parsing as just a number
					if _, err := fmt.Sscanf(collateralIndexStr, "%d", &collateralIndex); err != nil {
						continue // Skip invalid indices
					}
				}

				// Query individual collateral details
				denom, err := t.queryCollateralDenom(ctx, collateralIndex)
				if err != nil {
					continue
				}
				collaterals = append(collaterals, CollateralInfo{
					Index: collateralIndex,
					Denom: denom,
				})
			}
			return collaterals, nil
		}
	}

	// Fallback: try common indices (0-10) to find available collaterals
	var collaterals []CollateralInfo
	for i := uint64(0); i <= 10; i++ {
		denom, err := t.queryCollateralDenom(ctx, i)
		if err == nil && denom != "" {
			collaterals = append(collaterals, CollateralInfo{
				Index: i,
				Denom: denom,
			})
		}
	}

	return collaterals, nil
}

// CollateralInfo contains information about a collateral token
type CollateralInfo struct {
	Index uint64
	Denom string
}

// queryCollateralDenom queries the perp contract for the denomination of a collateral token by index
func (t *EVMTrader) queryCollateralDenom(ctx context.Context, collateralIndex uint64) (string, error) {
	queryMsg := map[string]interface{}{
		"get_collateral": map[string]interface{}{
			"index": collateralIndex,
		},
	}

	fmt.Println("queryMsg", queryMsg)

	// Execute the query using the helper method
	responseBytes, err := t.queryWasmContract(ctx, t.addrs.PerpAddress, queryMsg)
	if err != nil {
		return "", err
	}

	// Parse JSON response - get_collateral returns { denom: string, ... }
	var collateralResp struct {
		Denom string `json:"denom"`
	}
	if err := json.Unmarshal(responseBytes, &collateralResp); err != nil {
		return "", fmt.Errorf("unmarshal collateral response: %w, raw: %s", err, string(responseBytes))
	}

	if collateralResp.Denom == "" {
		return "", fmt.Errorf("denom field is empty in response, raw: %s", string(responseBytes))
	}

	return collateralResp.Denom, nil
}

// queryPairDepth queries the perp contract for pair depth information for a market
// Returns true if pair_depth exists, false if it doesn't
func (t *EVMTrader) queryPairDepth(ctx context.Context, marketIndex uint64) (bool, error) {
	queryMsg := map[string]interface{}{
		"get_pair_depth": map[string]interface{}{
			"index": marketIndex,
		},
	}

	// Execute the query using the helper method
	responseBytes, err := t.queryWasmContract(ctx, t.addrs.PerpAddress, queryMsg)
	if err != nil {
		// If the query fails, it likely means pair_depth doesn't exist
		return false, nil
	}

	// Try to parse JSON response - if it parses successfully, pair_depth exists
	var pairDepthResp struct {
		OnePercentDepthAboveUsd string `json:"one_percent_depth_above_usd"`
		OnePercentDepthBelowUsd string `json:"one_percent_depth_below_usd"`
		Exponent                string `json:"exponent"`
	}
	if err := json.Unmarshal(responseBytes, &pairDepthResp); err != nil {
		// If parsing fails, pair_depth doesn't exist
		return false, nil
	}

	return true, nil
}

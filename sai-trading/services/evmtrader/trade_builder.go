package evmtrader

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
)

// buildOpenTradeMessage builds the open_trade message from parameters
func (t *EVMTrader) buildOpenTradeMessage(params *OpenTradeParams) ([]byte, error) {
	openTradeMsgData := map[string]interface{}{
		"market_index":     fmt.Sprintf("MarketIndex(%d)", params.MarketIndex),
		"leverage":         strconv.FormatUint(params.Leverage, 10),
		"long":             params.Long,
		"collateral_index": fmt.Sprintf("TokenIndex(%d)", params.CollateralIndex),
		"trade_type":       params.TradeType,
		"slippage_p":       params.SlippageP,
		"is_evm_origin":    true, // Required when calling from EVM
	}

	// open_price is required by the contract
	if params.OpenPrice == nil {
		return nil, fmt.Errorf("open_price is required")
	}
	openTradeMsgData["open_price"] = strconv.FormatFloat(*params.OpenPrice, 'f', -1, 64)

	// Only set TP/SL if provided
	if params.TP != nil {
		openTradeMsgData["tp"] = strconv.FormatFloat(*params.TP, 'f', -1, 64)
	}
	if params.SL != nil {
		openTradeMsgData["sl"] = strconv.FormatFloat(*params.SL, 'f', -1, 64)
	}

	openTradeMsg := map[string]interface{}{
		"open_trade": openTradeMsgData,
	}

	return json.Marshal(openTradeMsg)
}

// OpenTradeFromJSON loads trade parameters from a JSON file and opens the trade
func (t *EVMTrader) OpenTradeFromJSON(ctx context.Context, jsonPath string) error {
	chainID, err := t.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("chain id: %w", err)
	}

	// Load JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read json file: %w", err)
	}

	var jsonMsg map[string]interface{}
	if err := json.Unmarshal(data, &jsonMsg); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}

	openTradeData, ok := jsonMsg["open_trade"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing 'open_trade' key in JSON")
	}

	// Parse parameters from JSON
	params, err := t.parseTradeParamsFromJSON(openTradeData)
	if err != nil {
		return fmt.Errorf("parse trade params: %w", err)
	}

	// Execute the trade
	return t.OpenTrade(ctx, chainID, params)
}

// parseTradeParamsFromJSON parses trade parameters from JSON data
func (t *EVMTrader) parseTradeParamsFromJSON(data map[string]interface{}) (*OpenTradeParams, error) {
	// Parse collateral_amount - required field
	var amt *big.Int
	ca, exists := data["collateral_amount"]
	if !exists || ca == nil {
		return nil, fmt.Errorf("collateral_amount is required in JSON")
	}

	switch v := ca.(type) {
	case string:
		if v == "" {
			return nil, fmt.Errorf("collateral_amount cannot be empty")
		}
		var ok bool
		amt, ok = new(big.Int).SetString(v, 10)
		if !ok {
			return nil, fmt.Errorf("parse collateral_amount: invalid number format '%s'", v)
		}
	case float64:
		amt = big.NewInt(int64(v))
	case int64:
		amt = big.NewInt(v)
	case int:
		amt = big.NewInt(int64(v))
	default:
		return nil, fmt.Errorf("parse collateral_amount: unsupported type, expected string or number, got %T", v)
	}

	if amt.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("collateral_amount must be positive, got: %v", ca)
	}

	params := &OpenTradeParams{
		CollateralAmt: amt,
		SlippageP:     "1",  // Default
		Long:          true, // Default
	}

	// Parse market_index
	if mi, ok := data["market_index"].(string); ok {
		idx, err := t.parseWrappedIndex(mi)
		if err != nil {
			return nil, fmt.Errorf("parse market_index: %w", err)
		}
		params.MarketIndex = uint64(idx)
	} else {
		params.MarketIndex = t.cfg.MarketIndex // Use config default
	}

	// Parse leverage
	if lev, ok := data["leverage"].(string); ok {
		l, err := strconv.ParseUint(lev, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse leverage: %w", err)
		}
		params.Leverage = l
	} else {
		return nil, fmt.Errorf("leverage is required")
	}

	// Parse long
	if long, ok := data["long"].(bool); ok {
		params.Long = long
	}

	// Parse collateral_index
	if ci, ok := data["collateral_index"].(string); ok {
		idx, err := t.parseWrappedIndex(ci)
		if err != nil {
			return nil, fmt.Errorf("parse collateral_index: %w", err)
		}
		params.CollateralIndex = uint64(idx)
	} else {
		params.CollateralIndex = t.cfg.CollateralIndex // Use config default
	}

	// Parse trade_type
	if tt, ok := data["trade_type"].(string); ok {
		params.TradeType = tt
	} else {
		params.TradeType = "trade" // Default
	}

	// Parse open_price (optional for market orders, required for limit/stop orders)
	// NOTE: For market trades, open_price is optional - the contract uses the execution price.
	// For limit/stop orders, open_price is required.
	if op, ok := data["open_price"].(string); ok && op != "" {
		price, err := strconv.ParseFloat(op, 64)
		if err != nil {
			return nil, fmt.Errorf("parse open_price: %w", err)
		}
		// Validate open_price is reasonable (not zero or negative)
		if price <= 0 {
			return nil, fmt.Errorf("open_price must be positive, got: %f", price)
		}
		params.OpenPrice = &price
	} else if params.TradeType != "trade" {
		// open_price is required for limit/stop orders
		return nil, fmt.Errorf("open_price is required for %s orders", params.TradeType)
	}
	// For market orders, open_price can be nil (optional)

	if params.TradeType != "trade" {
		// Only parse TP/SL for limit/stop orders
		if tp, ok := data["tp"].(string); ok && tp != "" && tp != "0" {
			price, err := strconv.ParseFloat(tp, 64)
			if err != nil {
				return nil, fmt.Errorf("parse tp: %w", err)
			}
			params.TP = &price
		}

		if sl, ok := data["sl"].(string); ok && sl != "" && sl != "0" {
			price, err := strconv.ParseFloat(sl, 64)
			if err != nil {
				return nil, fmt.Errorf("parse sl: %w", err)
			}
			params.SL = &price
		}
	}
	// For market trades, TP and SL are intentionally omitted (params.TP and params.SL remain nil)

	// Parse slippage_p (optional)
	if sp, ok := data["slippage_p"].(string); ok {
		params.SlippageP = sp
	}

	return params, nil
}

// buildCloseTradeMessage builds the close_trade_market message from trade index
func (t *EVMTrader) buildCloseTradeMessage(tradeIndex uint64) ([]byte, error) {
	closeTradeMsgData := map[string]interface{}{
		"trade_index": fmt.Sprintf("UserTradeIndex(%d)", tradeIndex),
	}

	closeTradeMsg := map[string]interface{}{
		"close_trade_market": closeTradeMsgData,
	}

	return json.Marshal(closeTradeMsg)
}
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
	// Validate required parameters according to specification
	if err := validateTradeParams(params); err != nil {
		return nil, fmt.Errorf("invalid trade parameters: %w", err)
	}

	openTradeMsgData := map[string]interface{}{
		"market_index":     fmt.Sprintf("MarketIndex(%d)", params.MarketIndex),
		"leverage":         strconv.FormatUint(params.Leverage, 10),
		"long":             params.Long,
		"collateral_index": fmt.Sprintf("TokenIndex(%d)", params.CollateralIndex),
		"trade_type":       params.TradeType,
		"slippage_p":       params.SlippageP,
		"is_evm_origin":    true, // Required when calling from EVM
	}

	// open_price is required by the contract for all trade types
	if params.OpenPrice == nil {
		return nil, fmt.Errorf("open_price is required")
	}
	// For limit/stop orders, open_price must be non-zero (trigger price)
	if isLimitOrStopOrder(params.TradeType) && *params.OpenPrice == 0 {
		return nil, fmt.Errorf("open_price must be non-zero for %s orders", params.TradeType)
	}
	openTradeMsgData["open_price"] = strconv.FormatFloat(*params.OpenPrice, 'f', -1, 64)

	// Only set TP/SL if explicitly provided by user
	// Note: The contract may set its own default TP/SL values for limit/stop orders
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

	// If open_price not provided for market orders, fetch from oracle
	if params.OpenPrice == nil && params.TradeType == TradeTypeMarket {
		price, err := t.fetchMarketPriceForIndex(ctx, params.MarketIndex)
		if err != nil {
			return fmt.Errorf("fetch market price for market %d: %w", params.MarketIndex, err)
		}
		t.log("Automatically fetched market price from oracle", "market_index", params.MarketIndex, "price", price)
		params.OpenPrice = &price
	}

	// Execute the trade
	return t.OpenTrade(ctx, chainID, params)
}

// parseTradeParamsFromJSON parses trade parameters from JSON data.
// This function orchestrates parsing of all fields and applies validation rules.
//
// Requirements:
//   - market_index: REQUIRED (no default)
//   - leverage: OPTIONAL (defaults to 1)
//   - long: OPTIONAL (defaults to true, meaning long position)
func (t *EVMTrader) parseTradeParamsFromJSON(data map[string]interface{}) (*OpenTradeParams, error) {
	// Initialize params with defaults
	params := &OpenTradeParams{
		SlippageP: defaultSlippagePercent,
		Long:      true,            // Default to long position
		Leverage:  1,               // Default leverage is 1x
		TradeType: TradeTypeMarket, // Default to market trade
	}

	// Parse collateral_amount (required)
	amt, err := parseCollateralAmount(data)
	if err != nil {
		return nil, err
	}
	params.CollateralAmt = amt

	// Parse market_index (REQUIRED - no default fallback)
	idx, err := t.parseRequiredIndexField(data, "market_index")
	if err != nil {
		return nil, err
	}
	params.MarketIndex = idx

	// Parse leverage (optional, defaults to 1)
	if leverage, err := parseOptionalPositiveUint64(data, "leverage"); err != nil {
		return nil, err
	} else if leverage != nil {
		params.Leverage = *leverage
	}
	// else: use default leverage of 1

	// Parse collateral_index (optional, falls back to config)
	if idx, err := t.parseOptionalIndexField(data, "collateral_index"); err != nil {
		return nil, err
	} else if idx != nil {
		params.CollateralIndex = *idx
	} else {
		params.CollateralIndex = t.cfg.CollateralIndex
	}

	// Parse long (optional, defaults to true for long position)
	// User should specify true for long, false for short
	if long, ok := data["long"].(bool); ok {
		params.Long = long
	}

	// Parse trade_type (optional, defaults to "trade")
	if tradeType, ok := data["trade_type"].(string); ok && tradeType != "" {
		params.TradeType = tradeType
	}

	// Parse slippage_p (optional, defaults to "1")
	if slippage, ok := data["slippage_p"].(string); ok && slippage != "" {
		params.SlippageP = slippage
	}

	// Parse open_price
	// - Market orders (trade): OPTIONAL - will be fetched from oracle if not provided
	// - Limit/Stop orders: REQUIRED - user must specify trigger price
	if err := t.parseOpenPrice(data, params); err != nil {
		return nil, err
	}

	// Parse TP/SL (only for limit/stop orders)
	if params.TradeType != TradeTypeMarket {
		if err := t.parseTakeProfitStopLoss(data, params); err != nil {
			return nil, err
		}
	}

	return params, nil
}

// parseCollateralAmount parses and validates the collateral_amount field from JSON.
// This is a required field that must be a positive integer.
func parseCollateralAmount(data map[string]interface{}) (*big.Int, error) {
	ca, exists := data["collateral_amount"]
	if !exists || ca == nil {
		return nil, fmt.Errorf("collateral_amount is required in JSON")
	}

	var amt *big.Int

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

	return amt, nil
}

// validateTradeParams validates trade parameters according to the specification.
// This is called before building the trade message to ensure all required fields are present.
// All trade types (Trade, Limit, Stop) require:
//   - leverage > 0
//   - open_price must be set by this point (fetched from oracle for market orders if needed)
//   - slippage_p is required
//   - collateral_amount > 0
func validateTradeParams(params *OpenTradeParams) error {
	// Validate leverage > 0 (required for all trade types)
	if params.Leverage == 0 {
		return fmt.Errorf("leverage must be greater than 0, got: %d", params.Leverage)
	}

	// Validate open_price is set (should be fetched by now if it was missing for market orders)
	if params.OpenPrice == nil {
		return fmt.Errorf("open_price must be set before building trade message (should be fetched from oracle for market orders)")
	}

	// For limit/stop orders, open_price must be non-zero (trigger price)
	if isLimitOrStopOrder(params.TradeType) && *params.OpenPrice == 0 {
		return fmt.Errorf("open_price must be non-zero for %s orders (trigger price)", params.TradeType)
	}

	// Validate slippage_p is set (required for all trade types)
	if params.SlippageP == "" {
		return fmt.Errorf("slippage_p is required")
	}

	// Validate collateral_amount is set and positive (required for all trade types)
	if params.CollateralAmt == nil || params.CollateralAmt.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("collateral_amount is required and must be positive")
	}

	return nil
}

// parseRequiredIndexField parses a required wrapped index field (e.g., "MarketIndex(0)").
// Returns an error if the field is missing or invalid.
func (t *EVMTrader) parseRequiredIndexField(data map[string]interface{}, fieldName string) (uint64, error) {
	value, ok := data[fieldName].(string)
	if !ok || value == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}

	idx, err := t.parseWrappedIndex(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", fieldName, err)
	}

	return uint64(idx), nil
}

// parseOptionalIndexField parses an optional wrapped index field (e.g., "MarketIndex(0)").
// Returns nil if the field doesn't exist, allowing caller to use default value.
func (t *EVMTrader) parseOptionalIndexField(data map[string]interface{}, fieldName string) (*uint64, error) {
	value, ok := data[fieldName].(string)
	if !ok || value == "" {
		return nil, nil // Field not present, use default
	}

	idx, err := t.parseWrappedIndex(value)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", fieldName, err)
	}

	result := uint64(idx)
	return &result, nil
}

// parseOpenPrice parses and validates the open_price field.
// Behavior by trade type:
//   - Market orders (trade): OPTIONAL - if not provided, will be fetched from oracle during execution
//   - Limit/Stop orders: REQUIRED - must be non-zero (trigger price)
func (t *EVMTrader) parseOpenPrice(data map[string]interface{}, params *OpenTradeParams) error {
	priceStr, ok := data["open_price"].(string)
	if !ok || priceStr == "" {
		// For limit/stop orders, open_price is REQUIRED (trigger price)
		if isLimitOrStopOrder(params.TradeType) {
			return fmt.Errorf("open_price is required for %s orders (trigger price)", params.TradeType)
		}
		// For market orders, open_price is OPTIONAL - will be fetched from oracle
		// Leave params.OpenPrice as nil, will be populated later
		return nil
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return fmt.Errorf("parse open_price: %w", err)
	}

	// For limit/stop orders, open_price must be non-zero (trigger price)
	if isLimitOrStopOrder(params.TradeType) && price == 0 {
		return fmt.Errorf("open_price must be non-zero for %s orders (trigger price)", params.TradeType)
	}

	// For all orders, price must be positive
	if price <= 0 {
		return fmt.Errorf("open_price must be positive, got: %f", price)
	}

	params.OpenPrice = &price
	return nil
}

// parseTakeProfitStopLoss parses the optional tp (take profit) and sl (stop loss) fields.
func (t *EVMTrader) parseTakeProfitStopLoss(data map[string]interface{}, params *OpenTradeParams) error {
	// Parse take profit (optional)
	if tpStr, ok := data["tp"].(string); ok && tpStr != "" && tpStr != "0" {
		tp, err := parsePositiveFloat(tpStr, "tp")
		if err != nil {
			return err
		}
		params.TP = &tp
	}

	// Parse stop loss (optional)
	if slStr, ok := data["sl"].(string); ok && slStr != "" && slStr != "0" {
		sl, err := parsePositiveFloat(slStr, "sl")
		if err != nil {
			return err
		}
		params.SL = &sl
	}

	return nil
}

// parsePositiveFloat parses a float value from a string and validates it's positive.
func parsePositiveFloat(value, fieldName string) (float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", fieldName, err)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive, got: %f", fieldName, parsed)
	}

	return parsed, nil
}

// parsePositiveUint64 parses a uint64 value from a string and validates it's greater than zero.
func parsePositiveUint64(value, fieldName string) (uint64, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", fieldName, err)
	}

	if parsed == 0 {
		return 0, fmt.Errorf("%s must be greater than 0, got: %d", fieldName, parsed)
	}

	return parsed, nil
}

// parseOptionalPositiveUint64 parses an optional uint64 field from JSON data.
// Returns nil if the field doesn't exist or is empty, allowing caller to use default value.
func parseOptionalPositiveUint64(data map[string]interface{}, fieldName string) (*uint64, error) {
	valStr, ok := data[fieldName].(string)
	if !ok || valStr == "" {
		return nil, nil // Field not present or empty, use default
	}

	val, err := parsePositiveUint64(valStr, fieldName)
	if err != nil {
		return nil, err
	}

	return &val, nil
}

// buildCloseTradeMessage builds the close_trade message from trade index
func (t *EVMTrader) buildCloseTradeMessage(tradeIndex uint64) ([]byte, error) {
	closeTradeMsgData := map[string]interface{}{
		"trade_index": fmt.Sprintf("UserTradeIndex(%d)", tradeIndex),
	}

	closeTradeMsg := map[string]interface{}{
		"close_trade": closeTradeMsgData,
	}

	return json.Marshal(closeTradeMsg)
}

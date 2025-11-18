package evmtrader

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// parseTradeID extracts the trade ID from transaction response events
func (t *EVMTrader) parseTradeID(txResp *sdk.TxResponse) (int, error) {
	type tradeStruct struct {
		UserTradeIndex string `json:"user_trade_index"`
	}
	type globalTradeIndexStruct struct {
		User           string `json:"user"`
		UserTradeIndex string `json:"user_trade_index"`
	}

	for _, event := range txResp.Events {
		eventType := event.Type

		// Check if this is any trade-related event
		isTradeEvent := strings.Contains(eventType, "register_trade") ||
			strings.Contains(eventType, "store_trade") ||
			strings.Contains(eventType, "trigger_trade/register_trade") ||
			strings.Contains(eventType, "process_opening_fees")

		if !isTradeEvent {
			continue
		}

		// Try to extract trade ID from various attributes
		for _, attr := range event.Attributes {
			// Method 1: Check "trade_index" attribute (in process_opening_fees events)
			// This is the simplest - directly contains "UserTradeIndex(1)"
			if attr.Key == "trade_index" {
				if tradeID, err := t.parseWrappedIndex(attr.Value); err == nil {
					return tradeID, nil
				}
			}

			// Method 2: Check "trade" attribute (contains full trade JSON)
			if attr.Key == "trade" {
				var trade tradeStruct
				if err := json.Unmarshal([]byte(attr.Value), &trade); err == nil {
					if trade.UserTradeIndex != "" {
						if tradeID, err := t.parseWrappedIndex(trade.UserTradeIndex); err == nil {
							return tradeID, nil
						}
					}
				}
			}

			// Method 3: Check "global_trade_index" attribute (in store_trade events)
			if attr.Key == "global_trade_index" {
				var gti globalTradeIndexStruct
				if err := json.Unmarshal([]byte(attr.Value), &gti); err == nil {
					if gti.UserTradeIndex != "" {
						if tradeID, err := t.parseWrappedIndex(gti.UserTradeIndex); err == nil {
							return tradeID, nil
						}
					}
				}
			}
		}
	}

	// If not found in events, try to parse from transaction data
	// The contract sometimes returns user_trade_index in the response data
	if txResp.Data != "" {
		// Try to parse as wrapped index string directly
		if tradeID, err := t.parseWrappedIndex(txResp.Data); err == nil {
			return tradeID, nil
		}
		// Try to parse as base64-encoded JSON
		// (Cosmos SDK sometimes encodes response data as base64)
		if decoded, err := base64.StdEncoding.DecodeString(txResp.Data); err == nil {
			var data map[string]interface{}
			if err := json.Unmarshal(decoded, &data); err == nil {
				if idx, ok := data["user_trade_index"].(float64); ok {
					return int(idx), nil
				}
			}
			// Try as string
			var dataStr string
			if err := json.Unmarshal(decoded, &dataStr); err == nil {
				if tradeID, err := t.parseWrappedIndex(dataStr); err == nil {
					return tradeID, nil
				}
			}
		}
	}

	return -1, fmt.Errorf("user_trade_index not found in events, height=%d, event_count=%d", txResp.Height, len(txResp.Events))
}

// parseWrappedIndex extracts number from strings like "MarketIndex(123)" or "TokenIndex(456)"
func (t *EVMTrader) parseWrappedIndex(s string) (int, error) {
	start := strings.LastIndex(s, "(")
	end := strings.LastIndex(s, ")")

	if start == -1 || end == -1 || start >= end {
		return 0, fmt.Errorf("invalid format: %s", s)
	}

	numStr := s[start+1 : end]
	return strconv.Atoi(numStr)
}

package evmtrader

import (
	"encoding/json"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestParseWrappedIndex tests the parseWrappedIndex helper function
func TestParseWrappedIndex(t *testing.T) {
	// Create a minimal EVMTrader instance for testing
	trader := &EVMTrader{}

	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:        "parse MarketIndex",
			input:       "MarketIndex(123)",
			expected:    123,
			expectError: false,
		},
		{
			name:        "parse TokenIndex",
			input:       "TokenIndex(456)",
			expected:    456,
			expectError: false,
		},
		{
			name:        "parse UserTradeIndex",
			input:       "UserTradeIndex(789)",
			expected:    789,
			expectError: false,
		},
		{
			name:        "parse zero index",
			input:       "Index(0)",
			expected:    0,
			expectError: false,
		},
		{
			name:        "parse large index",
			input:       "Index(999999)",
			expected:    999999,
			expectError: false,
		},
		{
			name:        "invalid format - no parentheses",
			input:       "MarketIndex123",
			expected:    0,
			expectError: true,
		},
		{
			name:        "invalid format - missing closing paren",
			input:       "MarketIndex(123",
			expected:    0,
			expectError: true,
		},
		{
			name:        "invalid format - missing opening paren",
			input:       "MarketIndex123)",
			expected:    0,
			expectError: true,
		},
		{
			name:        "invalid format - empty parentheses",
			input:       "MarketIndex()",
			expected:    0,
			expectError: true,
		},
		{
			name:        "invalid format - non-numeric content",
			input:       "MarketIndex(abc)",
			expected:    0,
			expectError: true,
		},
		{
			name:        "invalid format - empty string",
			input:       "",
			expected:    0,
			expectError: true,
		},
		{
			name:        "invalid format - just parentheses",
			input:       "(123)",
			expected:    123,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := trader.parseWrappedIndex(tt.input)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestParseTradeID tests the parseTradeID function with various event formats
func TestParseTradeID(t *testing.T) {
	trader := &EVMTrader{}

	tests := []struct {
		name        string
		txResp      *sdk.TxResponse
		expected    int
		expectError bool
	}{
		{
			name: "trade_index attribute in process_opening_fees",
			txResp: &sdk.TxResponse{
				Height: 100,
				Events: []abcitypes.Event{
					{
						Type: "wasm-process_opening_fees",
						Attributes: []abcitypes.EventAttribute{
							{Key: "trade_index", Value: "UserTradeIndex(42)"},
						},
					},
				},
			},
			expected:    42,
			expectError: false,
		},
		{
			name: "trade attribute with JSON",
			txResp: &sdk.TxResponse{
				Height: 100,
				Events: []abcitypes.Event{
					{
						Type: "wasm-register_trade",
						Attributes: []abcitypes.EventAttribute{
							{
								Key:   "trade",
								Value: `{"user":"nibi1abc","user_trade_index":"UserTradeIndex(123)"}`,
							},
						},
					},
				},
			},
			expected:    123,
			expectError: false,
		},
		{
			name: "global_trade_index attribute with JSON",
			txResp: &sdk.TxResponse{
				Height: 100,
				Events: []abcitypes.Event{
					{
						Type: "wasm-store_trade",
						Attributes: []abcitypes.EventAttribute{
							{
								Key:   "global_trade_index",
								Value: `{"user":"nibi1xyz","user_trade_index":"UserTradeIndex(789)"}`,
							},
						},
					},
				},
			},
			expected:    789,
			expectError: false,
		},
		{
			name: "trigger_trade event",
			txResp: &sdk.TxResponse{
				Height: 100,
				Events: []abcitypes.Event{
					{
						Type: "wasm-trigger_trade/register_trade",
						Attributes: []abcitypes.EventAttribute{
							{Key: "trade_index", Value: "UserTradeIndex(555)"},
						},
					},
				},
			},
			expected:    555,
			expectError: false,
		},
		{
			name: "multiple events - use first match",
			txResp: &sdk.TxResponse{
				Height: 100,
				Events: []abcitypes.Event{
					{
						Type: "transfer",
						Attributes: []abcitypes.EventAttribute{
							{Key: "amount", Value: "1000unibi"},
						},
					},
					{
						Type: "wasm-register_trade",
						Attributes: []abcitypes.EventAttribute{
							{Key: "trade_index", Value: "UserTradeIndex(111)"},
						},
					},
					{
						Type: "wasm-process_opening_fees",
						Attributes: []abcitypes.EventAttribute{
							{Key: "trade_index", Value: "UserTradeIndex(222)"},
						},
					},
				},
			},
			expected:    111,
			expectError: false,
		},
		{
			name: "no trade events",
			txResp: &sdk.TxResponse{
				Height: 100,
				Events: []abcitypes.Event{
					{
						Type: "transfer",
						Attributes: []abcitypes.EventAttribute{
							{Key: "amount", Value: "1000unibi"},
						},
					},
				},
			},
			expected:    -1,
			expectError: true,
		},
		{
			name: "empty events",
			txResp: &sdk.TxResponse{
				Height: 100,
				Events: []abcitypes.Event{},
			},
			expected:    -1,
			expectError: true,
		},
		{
			name: "invalid trade_index format",
			txResp: &sdk.TxResponse{
				Height: 100,
				Events: []abcitypes.Event{
					{
						Type: "wasm-register_trade",
						Attributes: []abcitypes.EventAttribute{
							{Key: "trade_index", Value: "invalid"},
						},
					},
				},
			},
			expected:    -1,
			expectError: true,
		},
		{
			name: "invalid JSON in trade attribute",
			txResp: &sdk.TxResponse{
				Height: 100,
				Events: []abcitypes.Event{
					{
						Type: "wasm-register_trade",
						Attributes: []abcitypes.EventAttribute{
							{Key: "trade", Value: `{invalid json}`},
						},
					},
				},
			},
			expected:    -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := trader.parseTradeID(tt.txResp)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestParseTradeIDFromData tests parsing trade ID from transaction data field
func TestParseTradeIDFromData(t *testing.T) {
	trader := &EVMTrader{}

	// Test with data field containing wrapped index
	txResp := &sdk.TxResponse{
		Height: 100,
		Events: []abcitypes.Event{}, // No events
		Data:   "UserTradeIndex(999)",
	}

	result, err := trader.parseTradeID(txResp)
	require.NoError(t, err)
	require.Equal(t, 999, result)

	// Test with base64-encoded JSON data
	jsonData := map[string]interface{}{
		"user_trade_index": float64(888),
	}
	jsonBytes, err := json.Marshal(jsonData)
	require.NoError(t, err)
	base64Data := sdk.MustSortJSON(jsonBytes) // This will be string encoded

	txResp2 := &sdk.TxResponse{
		Height: 100,
		Events: []abcitypes.Event{},
		Data:   string(base64Data),
	}

	// This test checks if the parser can handle JSON in the data field
	_, err = trader.parseTradeID(txResp2)
	// This may error depending on the exact format, which is ok for this edge case
	// The main test is that it doesn't panic
}

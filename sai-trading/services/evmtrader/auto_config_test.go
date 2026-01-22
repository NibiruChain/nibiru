package evmtrader_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/NibiruChain/nibiru/sai-trading/services/evmtrader"
	"github.com/stretchr/testify/require"
)

func TestLoadAutoTradingConfig(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		jsonContent string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			jsonContent: `{
				"network": {
					"mode": "localnet",
					"evm_rpc_url": "http://localhost:8545",
					"networks_toml": "networks.toml"
				},
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 1000000,
					"max_trade_size": 5000000,
					"min_leverage": 1,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 10,
					"max_open_positions": 5,
					"loop_delay_seconds": 30
				}
			}`,
			expectError: false,
		},
		{
			name: "valid configuration without network settings",
			jsonContent: `{
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 1000000,
					"max_trade_size": 5000000,
					"min_leverage": 1,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 10,
					"max_open_positions": 5,
					"loop_delay_seconds": 30
				}
			}`,
			expectError: false,
		},
		{
			name: "invalid JSON syntax",
			jsonContent: `{
				"trading": {
					"market_index": 0
					"collateral_index": 1
				}
			}`,
			expectError: true,
			errorMsg:    "parse config JSON",
		},
		{
			name: "min_trade_size greater than max_trade_size",
			jsonContent: `{
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 10000000,
					"max_trade_size": 5000000,
					"min_leverage": 1,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 10,
					"max_open_positions": 5,
					"loop_delay_seconds": 30
				}
			}`,
			expectError: true,
			errorMsg:    "min_trade_size",
		},
		{
			name: "min_leverage greater than max_leverage",
			jsonContent: `{
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 1000000,
					"max_trade_size": 5000000,
					"min_leverage": 20,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 10,
					"max_open_positions": 5,
					"loop_delay_seconds": 30
				}
			}`,
			expectError: true,
			errorMsg:    "min_leverage",
		},
		{
			name: "min_leverage is zero",
			jsonContent: `{
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 1000000,
					"max_trade_size": 5000000,
					"min_leverage": 0,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 10,
					"max_open_positions": 5,
					"loop_delay_seconds": 30
				}
			}`,
			expectError: true,
			errorMsg:    "min_leverage must be at least 1",
		},
		{
			name: "min_trade_size is zero",
			jsonContent: `{
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 0,
					"max_trade_size": 5000000,
					"min_leverage": 1,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 10,
					"max_open_positions": 5,
					"loop_delay_seconds": 30
				}
			}`,
			expectError: true,
			errorMsg:    "min_trade_size must be greater than 0",
		},
		{
			name: "blocks_before_close is zero",
			jsonContent: `{
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 1000000,
					"max_trade_size": 5000000,
					"min_leverage": 1,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 0,
					"max_open_positions": 5,
					"loop_delay_seconds": 30
				}
			}`,
			expectError: true,
			errorMsg:    "blocks_before_close must be greater than 0",
		},
		{
			name: "max_open_positions is zero",
			jsonContent: `{
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 1000000,
					"max_trade_size": 5000000,
					"min_leverage": 1,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 10,
					"max_open_positions": 0,
					"loop_delay_seconds": 30
				}
			}`,
			expectError: true,
			errorMsg:    "max_open_positions must be greater than 0",
		},
		{
			name: "loop_delay_seconds is zero",
			jsonContent: `{
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 1000000,
					"max_trade_size": 5000000,
					"min_leverage": 1,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 10,
					"max_open_positions": 5,
					"loop_delay_seconds": 0
				}
			}`,
			expectError: true,
			errorMsg:    "loop_delay_seconds must be greater than 0",
		},
		{
			name: "max_open_positions is negative",
			jsonContent: `{
				"trading": {
					"market_index": 0,
					"collateral_index": 1,
					"min_trade_size": 1000000,
					"max_trade_size": 5000000,
					"min_leverage": 1,
					"max_leverage": 10
				},
				"bot": {
					"blocks_before_close": 10,
					"max_open_positions": -1,
					"loop_delay_seconds": 30
				}
			}`,
			expectError: true,
			errorMsg:    "max_open_positions must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			configFile := filepath.Join(tmpDir, tt.name+".json")
			err := os.WriteFile(configFile, []byte(tt.jsonContent), 0644)
			require.NoError(t, err)

			// Load config
			cfg, err := evmtrader.LoadAutoTradingConfig(configFile)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
				require.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
			}
		})
	}
}

func TestLoadAutoTradingConfig_NonExistentFile(t *testing.T) {
	_, err := evmtrader.LoadAutoTradingConfig("/nonexistent/config.json")
	require.Error(t, err)
	require.Contains(t, err.Error(), "read config file")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      evmtrader.AutoTradingJSONConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MarketIndices:     []uint64{0},
					CollateralIndices: []uint64{1},
					MinTradeSize:      1000000,
					MaxTradeSize:      5000000,
					MinLeverage:       1,
					MaxLeverage:       10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  5,
					LoopDelaySeconds:  30,
				},
			},
			expectError: false,
		},
		{
			name: "min equals max is valid",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MarketIndices:     []uint64{0},
					CollateralIndices: []uint64{1},
					MinTradeSize:      5000000,
					MaxTradeSize:      5000000,
					MinLeverage:       5,
					MaxLeverage:       5,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  5,
					LoopDelaySeconds:  30,
				},
			},
			expectError: false,
		},
		{
			name: "min_trade_size > max_trade_size",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MinTradeSize: 10000000,
					MaxTradeSize: 5000000,
					MinLeverage:  1,
					MaxLeverage:  10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  5,
					LoopDelaySeconds:  30,
				},
			},
			expectError: true,
			errorMsg:    "min_trade_size (10000000) cannot be greater than max_trade_size (5000000)",
		},
		{
			name: "min_leverage > max_leverage",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MinTradeSize: 1000000,
					MaxTradeSize: 5000000,
					MinLeverage:  20,
					MaxLeverage:  10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  5,
					LoopDelaySeconds:  30,
				},
			},
			expectError: true,
			errorMsg:    "min_leverage (20) cannot be greater than max_leverage (10)",
		},
		{
			name: "min_leverage is zero",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MinTradeSize: 1000000,
					MaxTradeSize: 5000000,
					MinLeverage:  0,
					MaxLeverage:  10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  5,
					LoopDelaySeconds:  30,
				},
			},
			expectError: true,
			errorMsg:    "min_leverage must be at least 1",
		},
		{
			name: "min_trade_size is zero",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MinTradeSize: 0,
					MaxTradeSize: 5000000,
					MinLeverage:  1,
					MaxLeverage:  10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  5,
					LoopDelaySeconds:  30,
				},
			},
			expectError: true,
			errorMsg:    "min_trade_size must be greater than 0",
		},
		{
			name: "blocks_before_close is zero",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MinTradeSize: 1000000,
					MaxTradeSize: 5000000,
					MinLeverage:  1,
					MaxLeverage:  10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 0,
					MaxOpenPositions:  5,
					LoopDelaySeconds:  30,
				},
			},
			expectError: true,
			errorMsg:    "blocks_before_close must be greater than 0",
		},
		{
			name: "max_open_positions is zero",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MinTradeSize: 1000000,
					MaxTradeSize: 5000000,
					MinLeverage:  1,
					MaxLeverage:  10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  0,
					LoopDelaySeconds:  30,
				},
			},
			expectError: true,
			errorMsg:    "max_open_positions must be greater than 0",
		},
		{
			name: "loop_delay_seconds is zero",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MinTradeSize: 1000000,
					MaxTradeSize: 5000000,
					MinLeverage:  1,
					MaxLeverage:  10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  5,
					LoopDelaySeconds:  0,
				},
			},
			expectError: true,
			errorMsg:    "loop_delay_seconds must be greater than 0",
		},
		{
			name: "max_open_positions is negative",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MinTradeSize: 1000000,
					MaxTradeSize: 5000000,
					MinLeverage:  1,
					MaxLeverage:  10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  -5,
					LoopDelaySeconds:  30,
				},
			},
			expectError: true,
			errorMsg:    "max_open_positions must be greater than 0",
		},
		{
			name: "loop_delay_seconds is negative",
			config: evmtrader.AutoTradingJSONConfig{
				Trading: evmtrader.TradingSettings{
					MinTradeSize: 1000000,
					MaxTradeSize: 5000000,
					MinLeverage:  1,
					MaxLeverage:  10,
				},
				Bot: evmtrader.BotSettings{
					BlocksBeforeClose: 10,
					MaxOpenPositions:  5,
					LoopDelaySeconds:  -10,
				},
			},
			expectError: true,
			errorMsg:    "loop_delay_seconds must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Equal(t, tt.errorMsg, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestToAutoTradingConfig(t *testing.T) {
	jsonConfig := evmtrader.AutoTradingJSONConfig{
		Network: &evmtrader.NetworkSettings{
			Mode:         "testnet",
			EVMRPCUrl:    "https://evm-rpc.testnet.nibiru.fi",
			NetworksToml: "/path/to/networks.toml",
		},
		Trading: evmtrader.TradingSettings{
			MarketIndices:     []uint64{2},
			CollateralIndices: []uint64{3},
			MinTradeSize:      1000000,
			MaxTradeSize:      5000000,
			MinLeverage:       2,
			MaxLeverage:       15,
		},
		Bot: evmtrader.BotSettings{
			BlocksBeforeClose: 20,
			MaxOpenPositions:  10,
			LoopDelaySeconds:  60,
		},
	}

	result := jsonConfig.ToAutoTradingConfig()

	// Verify all fields are correctly mapped
	require.Equal(t, []uint64{2}, result.MarketIndices)
	require.Equal(t, []uint64{3}, result.CollateralIndices)
	require.Equal(t, uint64(1000000), result.MinTradeSize)
	require.Equal(t, uint64(5000000), result.MaxTradeSize)
	require.Equal(t, uint64(2), result.MinLeverage)
	require.Equal(t, uint64(15), result.MaxLeverage)
	require.Equal(t, uint64(20), result.BlocksBeforeClose)
	require.Equal(t, 10, result.MaxOpenPositions)
	require.Equal(t, 60, result.LoopDelaySeconds)
}

func TestToAutoTradingConfig_WithoutNetworkSettings(t *testing.T) {
	jsonConfig := evmtrader.AutoTradingJSONConfig{
		Network: nil, // No network settings
		Trading: evmtrader.TradingSettings{
			MarketIndices:     []uint64{0},
			CollateralIndices: []uint64{1},
			MinTradeSize:      500000,
			MaxTradeSize:      2000000,
			MinLeverage:       1,
			MaxLeverage:       5,
		},
		Bot: evmtrader.BotSettings{
			BlocksBeforeClose: 5,
			MaxOpenPositions:  3,
			LoopDelaySeconds:  15,
		},
	}

	result := jsonConfig.ToAutoTradingConfig()

	// Verify conversion works even without network settings
	require.Equal(t, []uint64{0}, result.MarketIndices)
	require.Equal(t, []uint64{1}, result.CollateralIndices)
	require.Equal(t, uint64(500000), result.MinTradeSize)
	require.Equal(t, uint64(2000000), result.MaxTradeSize)
	require.Equal(t, uint64(1), result.MinLeverage)
	require.Equal(t, uint64(5), result.MaxLeverage)
	require.Equal(t, uint64(5), result.BlocksBeforeClose)
	require.Equal(t, 3, result.MaxOpenPositions)
	require.Equal(t, 15, result.LoopDelaySeconds)
}

// TestTradingSettings_JSONUnmarshaling tests JSON unmarshaling with the new array format
func TestTradingSettings_JSONUnmarshaling(t *testing.T) {
	// Test new format (market_indices as array)
	jsonNew := `{
		"market_indices": [1, 2, 3],
		"collateral_indices": [0, 1],
		"min_trade_size": 1000000,
		"max_trade_size": 5000000,
		"min_leverage": 1,
		"max_leverage": 10
	}`

	var tsNew evmtrader.TradingSettings
	err := json.Unmarshal([]byte(jsonNew), &tsNew)
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2, 3}, tsNew.MarketIndices)
	require.Equal(t, []uint64{0, 1}, tsNew.CollateralIndices)
	require.Equal(t, uint64(1000000), tsNew.MinTradeSize)
	require.Equal(t, uint64(5000000), tsNew.MaxTradeSize)
	require.Equal(t, uint64(1), tsNew.MinLeverage)
	require.Equal(t, uint64(10), tsNew.MaxLeverage)

	// Test single value in array
	jsonSingle := `{
		"market_indices": [5],
		"collateral_indices": [2],
		"min_trade_size": 1000000,
		"max_trade_size": 5000000,
		"min_leverage": 1,
		"max_leverage": 10
	}`

	var tsSingle evmtrader.TradingSettings
	err = json.Unmarshal([]byte(jsonSingle), &tsSingle)
	require.NoError(t, err)
	require.Equal(t, []uint64{5}, tsSingle.MarketIndices)
	require.Equal(t, []uint64{2}, tsSingle.CollateralIndices)
}

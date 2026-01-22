package evmtrader

import (
	"encoding/json"
	"fmt"
	"os"
)

// AutoTradingJSONConfig represents the JSON configuration file structure
type AutoTradingJSONConfig struct {
	// Network settings (optional, can use command-line flags instead)
	Network *NetworkSettings `json:"network,omitempty"`

	// Trading parameters
	Trading TradingSettings `json:"trading"`

	// Bot behavior
	Bot BotSettings `json:"bot"`
}

// NetworkSettings contains network-related configuration
type NetworkSettings struct {
	Mode         string `json:"mode"`          // "localnet", "testnet", "mainnet"
	EVMRPCUrl    string `json:"evm_rpc_url"`   // Optional override
	NetworksToml string `json:"networks_toml"` // Path to networks.toml
}

// TradingSettings contains trading strategy parameters
type TradingSettings struct {
	MarketIndices     []uint64 `json:"market_indices,omitempty"`     // Array of market indices to randomly select from
	CollateralIndices []uint64 `json:"collateral_indices,omitempty"` // Array of collateral indices to randomly select from
	MinTradeSize      uint64   `json:"min_trade_size"`
	MaxTradeSize      uint64   `json:"max_trade_size"`
	MinLeverage       uint64   `json:"min_leverage"`
	MaxLeverage       uint64   `json:"max_leverage"`
}

// BotSettings contains bot behavior parameters
type BotSettings struct {
	BlocksBeforeClose uint64 `json:"blocks_before_close"`
	MaxOpenPositions  int    `json:"max_open_positions"`
	LoopDelaySeconds  int    `json:"loop_delay_seconds"`
}

// LoadAutoTradingConfig loads the auto-trading configuration from a JSON file
func LoadAutoTradingConfig(configPath string) (*AutoTradingJSONConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg AutoTradingJSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config JSON: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (cfg *AutoTradingJSONConfig) Validate() error {
	if len(cfg.Trading.MarketIndices) == 0 {
		return fmt.Errorf("at least one market index must be specified (use market_indices array)")
	}

	if len(cfg.Trading.CollateralIndices) == 0 {
		return fmt.Errorf("at least one collateral index must be specified (use collateral_indices array)")
	}

	// Validate trading settings
	if cfg.Trading.MinTradeSize > cfg.Trading.MaxTradeSize {
		return fmt.Errorf("min_trade_size (%d) cannot be greater than max_trade_size (%d)",
			cfg.Trading.MinTradeSize, cfg.Trading.MaxTradeSize)
	}
	if cfg.Trading.MinLeverage > cfg.Trading.MaxLeverage {
		return fmt.Errorf("min_leverage (%d) cannot be greater than max_leverage (%d)",
			cfg.Trading.MinLeverage, cfg.Trading.MaxLeverage)
	}
	if cfg.Trading.MinLeverage == 0 {
		return fmt.Errorf("min_leverage must be at least 1")
	}
	if cfg.Trading.MinTradeSize == 0 {
		return fmt.Errorf("min_trade_size must be greater than 0")
	}

	// Validate bot settings
	if cfg.Bot.BlocksBeforeClose == 0 {
		return fmt.Errorf("blocks_before_close must be greater than 0")
	}
	if cfg.Bot.MaxOpenPositions <= 0 {
		return fmt.Errorf("max_open_positions must be greater than 0")
	}
	if cfg.Bot.LoopDelaySeconds <= 0 {
		return fmt.Errorf("loop_delay_seconds must be greater than 0")
	}

	return nil
}

// ToAutoTradingConfig converts JSON config to AutoTradingConfig
func (cfg *AutoTradingJSONConfig) ToAutoTradingConfig() AutoTradingConfig {
	return AutoTradingConfig{
		MarketIndices:     cfg.Trading.MarketIndices,
		CollateralIndices: cfg.Trading.CollateralIndices,
		MinTradeSize:      cfg.Trading.MinTradeSize,
		MaxTradeSize:      cfg.Trading.MaxTradeSize,
		MinLeverage:       cfg.Trading.MinLeverage,
		MaxLeverage:       cfg.Trading.MaxLeverage,
		BlocksBeforeClose: cfg.Bot.BlocksBeforeClose,
		MaxOpenPositions:  cfg.Bot.MaxOpenPositions,
		LoopDelaySeconds:  cfg.Bot.LoopDelaySeconds,
	}
}

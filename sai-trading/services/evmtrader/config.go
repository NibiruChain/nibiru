package evmtrader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config holds runtime configuration for the EVM trader service.
type Config struct {
	// Network and contracts
	EVMRPCUrl        string
	GrpcUrl          string
	ChainID          string
	ContractsEnvFile string
	// ContractAddresses can be set directly (from TOML) or loaded from ContractsEnvFile
	ContractAddresses *ContractAddresses

	// Account
	PrivateKeyHex string
	Mnemonic      string
	CosmosAddress string

	// Notifications
	SlackWebhook      string
	SlackErrorFilters *ErrorFilters // Error filtering configuration (nil = send all)

	// Strategy
	TradeSize        uint64 // Exact trade size (if set, overrides min/max)
	TradeSizeMin     uint64
	TradeSizeMax     uint64
	Leverage         uint64 // Exact leverage (if set, overrides min/max)
	LeverageMin      uint64
	LeverageMax      uint64
	Long             *bool    // Trade direction: true for long, false for short, nil for random
	OpenPrice        *float64 // Open price (optional for market orders, required for limit/stop)
	MarketIndex      uint64
	CollateralIndex  uint64
	TradeType        string
	EnableLimitOrder bool

	// Static JSON file for trades
	TradeJSONFile string // Path to JSON file with open_trade parameters
}

// ContractAddresses stores addresses/ids loaded from localnet_contracts.env
type ContractAddresses struct {
	OracleAddress string
	PerpAddress   string
	VaultAddress  string
}

// ErrorFilters defines include/exclude keyword filters for Slack notifications
type ErrorFilters struct {
	Include []string `toml:"include"` // Only send errors containing these keywords (empty = send all)
	Exclude []string `toml:"exclude"` // Never send errors containing these keywords
}

// NetworkConfig represents the TOML configuration for all networks
type NetworkConfig struct {
	Localnet      NetworkInfo        `toml:"localnet"`
	Testnet       NetworkInfo        `toml:"testnet"`
	Mainnet       NetworkInfo        `toml:"mainnet"`
	Notifications NotificationConfig `toml:"notifications"`
}

// NotificationConfig contains notification filter settings
type NotificationConfig struct {
	Filters ErrorFilters `toml:"filters"`
}

// NetworkInfo contains configuration for a specific network
type NetworkInfo struct {
	Name      string         `toml:"name"`
	EVMRPCUrl string         `toml:"evm_rpc_url"`
	GrpcUrl   string         `toml:"grpc_url"`
	ChainID   string         `toml:"chain_id"`
	Contracts ContractConfig `toml:"contracts"`
}

// ContractConfig contains contract addresses
type ContractConfig struct {
	OracleAddress string `toml:"oracle_address"`
	PerpAddress   string `toml:"perp_address"`
}

// loadContractAddresses reads a simple KEY=VALUE env file.
func loadContractAddresses(envFile string) (ContractAddresses, error) {
	data, err := os.ReadFile(envFile)
	if err != nil {
		return ContractAddresses{}, fmt.Errorf("read env file: %w", err)
	}
	var addrs ContractAddresses
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.Trim(strings.TrimSpace(kv[1]), "\"")
		switch key {
		case "ORACLE_ADDRESS":
			addrs.OracleAddress = val
		case "PERP_ADDRESS":
			addrs.PerpAddress = val
		case "VAULT_ADDRESS":
			addrs.VaultAddress = val
		}
	}
	return addrs, nil
}

// LoadNetworkConfig loads network configuration from TOML file
func LoadNetworkConfig(tomlFile string) (NetworkConfig, error) {
	data, err := os.ReadFile(tomlFile)
	if err != nil {
		return NetworkConfig{}, fmt.Errorf("read TOML file: %w", err)
	}

	var config NetworkConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return NetworkConfig{}, fmt.Errorf("parse TOML: %w", err)
	}

	return config, nil
}

// GetNetworkInfo returns the NetworkInfo for a given network mode
func GetNetworkInfo(config NetworkConfig, networkMode string) (*NetworkInfo, error) {
	switch networkMode {
	case "localnet":
		return &config.Localnet, nil
	case "testnet":
		return &config.Testnet, nil
	case "mainnet":
		return &config.Mainnet, nil
	default:
		return nil, fmt.Errorf("unknown network mode: %s", networkMode)
	}
}

// ContractAddressesFromNetworkInfo converts NetworkInfo to ContractAddresses
func ContractAddressesFromNetworkInfo(netInfo *NetworkInfo) ContractAddresses {
	return ContractAddresses{
		OracleAddress: netInfo.Contracts.OracleAddress,
		PerpAddress:   netInfo.Contracts.PerpAddress,
	}
}

// normalizeConfigPaths normalizes file paths in config to be resilient to different CWDs.
func normalizeConfigPaths(cfg *Config) {
	if !filepath.IsAbs(cfg.ContractsEnvFile) && cfg.ContractsEnvFile != "" {
		if abs, err := filepath.Abs(cfg.ContractsEnvFile); err == nil {
			cfg.ContractsEnvFile = abs
		}
	}
}

// setConfigDefaults sets default values for config fields.
func setConfigDefaults(cfg *Config) {
	if cfg.GrpcUrl == "" {
		cfg.GrpcUrl = "localhost:9090"
	}
	if cfg.ChainID == "" {
		cfg.ChainID = "nibiru-localnet-0"
	}
	if cfg.ContractsEnvFile == "" {
		// default local path with a fallback
		cfg.ContractsEnvFile = filepath.Join(".cache", "localnet_contracts.env")
		if _, err := os.Stat(cfg.ContractsEnvFile); os.IsNotExist(err) {
			cfg.ContractsEnvFile = "localnet_contracts.env"
		}
	}
}

package evmtrader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds runtime configuration for the EVM trader service.
type Config struct {
	// Network and contracts
	EVMRPCUrl        string
	GrpcUrl          string
	ChainID          string
	ContractsEnvFile string

	// Account
	PrivateKeyHex string

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
	OracleAddress    string
	PerpAddress      string
	VaultAddress     string
	TokenStNIBIERC20 string
	StNIBIDenom      string
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
		case "TOKEN_STNIBI":
			addrs.TokenStNIBIERC20 = val
		case "STNIBI":
			addrs.StNIBIDenom = val
		}
	}
	return addrs, nil
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

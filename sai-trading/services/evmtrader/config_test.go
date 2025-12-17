package evmtrader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NibiruChain/nibiru/sai-trading/services/evmtrader"
	"github.com/stretchr/testify/require"
)

func TestGetNetworkInfo(t *testing.T) {
	// Create a test network config
	config := evmtrader.NetworkConfig{
		Localnet: evmtrader.NetworkInfo{
			Name:      "localnet",
			EVMRPCUrl: "http://localhost:8545",
			GrpcUrl:   "localhost:9090",
			ChainID:   "nibiru-localnet-0",
		},
		Testnet: evmtrader.NetworkInfo{
			Name:      "testnet",
			EVMRPCUrl: "https://evm-rpc.testnet-1.nibiru.fi",
			GrpcUrl:   "grpc.testnet-1.nibiru.fi:443",
			ChainID:   "nibiru-testnet-1",
		},
		Mainnet: evmtrader.NetworkInfo{
			Name:      "mainnet",
			EVMRPCUrl: "https://evm-rpc.nibiru.fi",
			GrpcUrl:   "grpc.nibiru.fi:443",
			ChainID:   "cataclysm-1",
		},
	}

	tests := []struct {
		name        string
		networkMode string
		expectError bool
		expected    *evmtrader.NetworkInfo
	}{
		{
			name:        "localnet returns correct config",
			networkMode: "localnet",
			expectError: false,
			expected:    &config.Localnet,
		},
		{
			name:        "testnet returns correct config",
			networkMode: "testnet",
			expectError: false,
			expected:    &config.Testnet,
		},
		{
			name:        "mainnet returns correct config",
			networkMode: "mainnet",
			expectError: false,
			expected:    &config.Mainnet,
		},
		{
			name:        "invalid network mode returns error",
			networkMode: "invalid",
			expectError: true,
			expected:    nil,
		},
		{
			name:        "empty network mode returns error",
			networkMode: "",
			expectError: true,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evmtrader.GetNetworkInfo(config, tt.networkMode)
			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestContractAddressesFromNetworkInfo(t *testing.T) {
	netInfo := &evmtrader.NetworkInfo{
		Name:      "testnet",
		EVMRPCUrl: "https://evm-rpc.testnet-1.nibiru.fi",
		GrpcUrl:   "grpc.testnet-1.nibiru.fi:443",
		ChainID:   "nibiru-testnet-1",
		Contracts: evmtrader.ContractConfig{
			OracleAddress: "0x1111111111111111111111111111111111111111",
			PerpAddress:   "0x2222222222222222222222222222222222222222",
			VaultAddress:  "0x3333333333333333333333333333333333333333",
			EVMInterface:  "0x4444444444444444444444444444444444444444",
		},
		Tokens: evmtrader.TokenConfig{
			USDCEvm:     "0x5555555555555555555555555555555555555555",
			StNIBIEvm:   "0x6666666666666666666666666666666666666666",
			StNIBIDenom: "factory/nibi1abc/stnibi",
		},
	}

	result := evmtrader.ContractAddressesFromNetworkInfo(netInfo)

	require.Equal(t, "0x1111111111111111111111111111111111111111", result.OracleAddress)
	require.Equal(t, "0x2222222222222222222222222222222222222222", result.PerpAddress)
	require.Equal(t, "0x3333333333333333333333333333333333333333", result.VaultAddress)
	require.Equal(t, "0x6666666666666666666666666666666666666666", result.TokenStNIBIERC20)
	require.Equal(t, "factory/nibi1abc/stnibi", result.StNIBIDenom)
}

func TestLoadNetworkConfig(t *testing.T) {
	// Create a temporary TOML file for testing
	tmpDir := t.TempDir()
	tomlFile := filepath.Join(tmpDir, "networks.toml")

	tomlContent := `
[localnet]
name = "localnet"
evm_rpc_url = "http://localhost:8545"
grpc_url = "localhost:9090"
chain_id = "nibiru-localnet-0"

[localnet.contracts]
oracle_address = "0xOracle"
perp_address = "0xPerp"
vault_address = "0xVault"
evm_interface = "0xEVM"

[localnet.tokens]
usdc_evm = "0xUSDC"
stnibi_evm = "0xStNIBI"
stnibi_denom = "factory/nibi1abc/stnibi"

[testnet]
name = "testnet"
evm_rpc_url = "https://evm-rpc.testnet-1.nibiru.fi"
grpc_url = "grpc.testnet-1.nibiru.fi:443"
chain_id = "nibiru-testnet-1"

[testnet.contracts]
oracle_address = "0xTestOracle"
perp_address = "0xTestPerp"
vault_address = "0xTestVault"
evm_interface = "0xTestEVM"

[testnet.tokens]
usdc_evm = "0xTestUSDC"
stnibi_evm = "0xTestStNIBI"
stnibi_denom = "factory/nibi1test/stnibi"

[mainnet]
name = "mainnet"
evm_rpc_url = "https://evm-rpc.nibiru.fi"
grpc_url = "grpc.nibiru.fi:443"
chain_id = "cataclysm-1"

[mainnet.contracts]
oracle_address = "0xMainOracle"
perp_address = "0xMainPerp"
vault_address = "0xMainVault"
evm_interface = "0xMainEVM"

[mainnet.tokens]
usdc_evm = "0xMainUSDC"
stnibi_evm = "0xMainStNIBI"
stnibi_denom = "unibi"

[notifications.filters]
include = ["critical", "error"]
exclude = ["debug"]
`

	err := os.WriteFile(tomlFile, []byte(tomlContent), 0644)
	require.NoError(t, err)

	// Test loading the config
	config, err := evmtrader.LoadNetworkConfig(tomlFile)
	require.NoError(t, err)

	// Verify localnet
	require.Equal(t, "localnet", config.Localnet.Name)
	require.Equal(t, "http://localhost:8545", config.Localnet.EVMRPCUrl)
	require.Equal(t, "localhost:9090", config.Localnet.GrpcUrl)
	require.Equal(t, "nibiru-localnet-0", config.Localnet.ChainID)
	require.Equal(t, "0xOracle", config.Localnet.Contracts.OracleAddress)
	require.Equal(t, "0xPerp", config.Localnet.Contracts.PerpAddress)
	require.Equal(t, "0xVault", config.Localnet.Contracts.VaultAddress)
	require.Equal(t, "0xUSDC", config.Localnet.Tokens.USDCEvm)
	require.Equal(t, "0xStNIBI", config.Localnet.Tokens.StNIBIEvm)
	require.Equal(t, "factory/nibi1abc/stnibi", config.Localnet.Tokens.StNIBIDenom)

	// Verify testnet
	require.Equal(t, "testnet", config.Testnet.Name)
	require.Equal(t, "https://evm-rpc.testnet-1.nibiru.fi", config.Testnet.EVMRPCUrl)
	require.Equal(t, "0xTestOracle", config.Testnet.Contracts.OracleAddress)

	// Verify mainnet
	require.Equal(t, "mainnet", config.Mainnet.Name)
	require.Equal(t, "https://evm-rpc.nibiru.fi", config.Mainnet.EVMRPCUrl)
	require.Equal(t, "0xMainOracle", config.Mainnet.Contracts.OracleAddress)

	// Verify notifications
	require.Equal(t, []string{"critical", "error"}, config.Notifications.Filters.Include)
	require.Equal(t, []string{"debug"}, config.Notifications.Filters.Exclude)
}

func TestLoadNetworkConfig_NonExistentFile(t *testing.T) {
	_, err := evmtrader.LoadNetworkConfig("/nonexistent/file.toml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "read TOML file")
}

func TestLoadNetworkConfig_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	tomlFile := filepath.Join(tmpDir, "invalid.toml")

	// Write invalid TOML
	err := os.WriteFile(tomlFile, []byte("this is not valid toml {{{"), 0644)
	require.NoError(t, err)

	_, err = evmtrader.LoadNetworkConfig(tomlFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse TOML")
}

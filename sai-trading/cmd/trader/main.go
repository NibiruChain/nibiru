package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/NibiruChain/nibiru/sai-trading/services/evmtrader"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	// Network config
	evmRPCUrl        string
	contractsEnvFile string
	networkMode      string

	// Account
	privateKeyHex string
	mnemonic      string

	// Strategy config
	tradeSizeMin     uint64
	tradeSizeMax     uint64
	leverageMin      uint64
	leverageMax      uint64
	marketIndex      uint64
	collateralIndex  uint64
	enableLimitOrder bool

	// Static JSON file
	tradeJSONFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "trader",
		Short: "Run EVM-based trading bot for Sai Perps",
		Long: `EVM-based trading service that performs end-to-end trading operations
as a liveness monitor with detailed logs for debugging and diagnostics.

This service uses the EVM precompile interface to interact with Sai Perps contracts.`,
		RunE: runTrader,
	}

	// Network flags
	rootCmd.Flags().StringVar(&networkMode, "network", "localnet", "Network mode: localnet, testnet, or mainnet")
	rootCmd.Flags().StringVar(&evmRPCUrl, "evm-rpc", "", "EVM RPC URL (overrides network mode default)")
	rootCmd.Flags().StringVar(&contractsEnvFile, "contracts-env", "", "Path to contracts env file (auto-detected if not set)")

	// Account flags
	rootCmd.Flags().StringVar(&privateKeyHex, "private-key", "", "Private key in hex format (or set EVM_PRIVATE_KEY env var)")
	rootCmd.Flags().StringVar(&mnemonic, "mnemonic", "", "BIP39 mnemonic phrase (or set EVM_MNEMONIC env var)")

	// Strategy flags
	rootCmd.Flags().Uint64Var(&tradeSizeMin, "trade-size-min", 10_000, "Minimum trade size in smallest units")
	rootCmd.Flags().Uint64Var(&tradeSizeMax, "trade-size-max", 50_000, "Maximum trade size in smallest units")
	rootCmd.Flags().Uint64Var(&leverageMin, "leverage-min", 5, "Minimum leverage (e.g., 5 for 5x)")
	rootCmd.Flags().Uint64Var(&leverageMax, "leverage-max", 20, "Maximum leverage (e.g., 20 for 20x)")
	rootCmd.Flags().Uint64Var(&marketIndex, "market-index", 0, "Market index to trade")
	rootCmd.Flags().Uint64Var(&collateralIndex, "collateral-index", 1, "Collateral token index")
	rootCmd.Flags().BoolVar(&enableLimitOrder, "enable-limit-order", false, "Enable limit order trading")
	rootCmd.Flags().StringVar(&tradeJSONFile, "trade-json", "", "Path to JSON file with open_trade parameters (overrides dynamic trading)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runTrader(cmd *cobra.Command, args []string) error {
	// Load .env file if it exists (best effort - ignore errors)
	_ = godotenv.Load(".env")

	cfg := evmtrader.Config{}

	// Set network defaults if not overridden
	var grpcUrl, chainID string
	if evmRPCUrl == "" {
		switch networkMode {
		case "localnet":
			evmRPCUrl = "http://localhost:8545"
			grpcUrl = "localhost:9090"
			chainID = "nibiru-localnet-0"
		// case "testnet":
		// 	evmRPCUrl = "https://evm-rpc.testnet-2.nibiru.fi"
		// 	grpcUrl = "grpc.testnet-2.nibiru.fi:443"
		// 	chainID = "nibiru-testnet-2"
		// case "mainnet":
		// 	evmRPCUrl = "https://evm-rpc.nibiru.fi"
		// 	grpcUrl = "grpc.nibiru.fi:443"
		// 	chainID = "nibiru-mainnet-1"
		default:
			return fmt.Errorf("unknown network mode: %s (use: localnet, testnet, mainnet)", networkMode)
		}
	} else {
		// If EVM RPC is set but gRPC/ChainID aren't, use localnet defaults
		if grpcUrl == "" {
			grpcUrl = "localhost:9090"
		}
		if chainID == "" {
			chainID = "nibiru-localnet-0"
		}
	}
	cfg.EVMRPCUrl = evmRPCUrl
	cfg.GrpcUrl = grpcUrl
	cfg.ChainID = chainID

	// Auto-detect contracts env file if not provided
	if contractsEnvFile == "" {
		contractsEnvFile = detectContractsEnvFile(networkMode)
	}
	cfg.ContractsEnvFile = contractsEnvFile

	// Get private key or mnemonic: try flags first, then env, then prompt if allowed
	if privateKeyHex == "" {
		privateKeyHex = os.Getenv("EVM_PRIVATE_KEY")
	}
	if mnemonic == "" {
		mnemonic = os.Getenv("EVM_MNEMONIC")
	}

	// If we have a mnemonic, convert it to private key using Nibiru's built-in EVM HD derivation
	if mnemonic != "" {
		privKeyHex, err := mnemonicToPrivateKeyHex(mnemonic)
		if err != nil {
			return fmt.Errorf("failed to convert mnemonic to private key: %w", err)
		}
		privateKeyHex = privKeyHex
	}

	if privateKeyHex == "" {
		return fmt.Errorf("private key or mnemonic required: set --private-key or --mnemonic flag, EVM_PRIVATE_KEY or EVM_MNEMONIC env var, or use --prompt-key")
	}
	cfg.PrivateKeyHex = privateKeyHex

	// Strategy config
	cfg.TradeSizeMin = tradeSizeMin
	cfg.TradeSizeMax = tradeSizeMax
	cfg.LeverageMin = leverageMin
	cfg.LeverageMax = leverageMax
	cfg.MarketIndex = marketIndex
	cfg.CollateralIndex = collateralIndex
	cfg.EnableLimitOrder = enableLimitOrder
	cfg.TradeJSONFile = tradeJSONFile

	// Validate config
	if cfg.TradeSizeMax > 0 && cfg.TradeSizeMin >= cfg.TradeSizeMax {
		return fmt.Errorf("trade-size-min must be less than trade-size-max")
	}
	if cfg.LeverageMax > 0 && cfg.LeverageMin >= cfg.LeverageMax {
		return fmt.Errorf("leverage-min must be less than leverage-max")
	}

	// Create trader
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trader, err := evmtrader.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create trader: %w", err)
	}
	defer trader.Close()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run trader in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- trader.Run(ctx)
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		fmt.Fprintf(os.Stderr, "\nReceived signal: %v, shutting down...\n", sig)
		cancel()
		// Wait for trader to finish
		if err := <-errChan; err != nil && err != context.Canceled {
			return err
		}
		return nil
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("trader error: %w", err)
		}
		return nil
	}
}

// detectContractsEnvFile tries common locations for the contracts env file.
func detectContractsEnvFile(networkMode string) string {
	// Try localnet paths first
	candidates := []string{
		filepath.Join(".cache", "localnet_contracts.env"),
		"localnet_contracts.env",
		filepath.Join("sai-perps", "scripts", "e2e_test", "localnet_contracts.env"),
	}

	// Add network-specific paths if not localnet
	if networkMode != "localnet" {
		candidates = append(candidates,
			fmt.Sprintf("%s_contracts.env", networkMode),
			filepath.Join(".cache", fmt.Sprintf("%s_contracts.env", networkMode)),
		)
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Return default even if it doesn't exist (will error later with better message)
	return candidates[0]
}

func mnemonicToPrivateKeyHex(mnemonic string) (string, error) {
	privKeyBytes, err := hd.EthSecp256k1.Derive()(mnemonic, keyring.DefaultBIP39Passphrase, eth.BIP44HDPath)
	if err != nil {
		return "", fmt.Errorf("derive private key: %w", err)
	}

	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		return "", fmt.Errorf("convert to ECDSA: %w", err)
	}

	return fmt.Sprintf("%x", crypto.FromECDSA(privKey)), nil
}

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NibiruChain/nibiru/sai-trading/services/evmtrader"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	// Network config (shared across subcommands)
	evmRPCUrl        string
	contractsEnvFile string
	networksTomlFile string
	networkMode      string

	// Account (shared across subcommands)
	privateKeyHex string
	mnemonic      string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "trader",
		Short: "EVM-based trading commands for Sai Perps",
		Long: `EVM-based trading service that performs trading operations using the EVM precompile interface
to interact with Sai Perps contracts.`,
	}

	// Shared flags for all subcommands
	rootCmd.PersistentFlags().StringVar(&networkMode, "network", "localnet", "Network mode: localnet, testnet, or mainnet")
	rootCmd.PersistentFlags().StringVar(&evmRPCUrl, "evm-rpc", "", "EVM RPC URL (overrides network mode default)")
	rootCmd.PersistentFlags().StringVar(&networksTomlFile, "networks-toml", "networks.toml", "Path to networks TOML configuration file")
	rootCmd.PersistentFlags().StringVar(&contractsEnvFile, "contracts-env", "", "Path to contracts env file (legacy, overrides networks.toml)")
	rootCmd.PersistentFlags().StringVar(&privateKeyHex, "private-key", "", "Private key in hex format (or set EVM_PRIVATE_KEY env var)")
	rootCmd.PersistentFlags().StringVar(&mnemonic, "mnemonic", "", "BIP39 mnemonic phrase (or set EVM_MNEMONIC env var)")

	// Add subcommands
	rootCmd.AddCommand(newOpenCmd())
	rootCmd.AddCommand(newCloseCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newPositionsCmd())
	rootCmd.AddCommand(newAutoCmd())

	// Default to open command for backward compatibility
	rootCmd.RunE = newOpenCmd().RunE

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// newOpenCmd creates the open subcommand
func newOpenCmd() *cobra.Command {
	var (
		// Strategy config
		tradeSize       uint64
		leverage        uint64
		long            bool
		marketIndex     uint64
		collateralIndex uint64
		tradeType       string
		openPrice       float64
		tradeJSONFile   string
	)

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open a trade in Sai Perps",
		Long: `Open a trade (market order, limit order, or stop order) in Sai Perps.

This command opens a new trading position using the EVM precompile interface.

Examples:
  # Market order (auto-fetch price)
  trader open --market-index 0 --leverage 5 --long true

  # Limit order with explicit trigger price
  trader open --trade-type limit --market-index 0 --open-price 70000 --long

  # Limit order with auto-fetch price (uses oracle price as-is)
  trader open --trade-type limit --market-index 0 --leverage 5 --long

  # Short position
  trader open --market-index 0 --long=false

  # Using JSON file
  trader open --trade-json sample_txs/open_trade.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var longPtr *bool
			if cmd.Flags().Changed("long") {
				longPtr = &long
			}
			var openPricePtr *float64
			if cmd.Flags().Changed("open-price") {
				openPricePtr = &openPrice
			}
			return runOpen(tradeSize, leverage, longPtr, marketIndex, collateralIndex, tradeType, openPricePtr, tradeJSONFile)
		},
	}

	// Strategy flags - exact values (override ranges if set)
	cmd.Flags().Uint64Var(&tradeSize, "trade-size", 0, "Exact trade size in smallest units (overrides min/max)")
	cmd.Flags().Uint64Var(&leverage, "leverage", 0, "Exact leverage (e.g., 10 for 10x, default: 1)")
	cmd.Flags().BoolVar(&long, "long", false, "Trade direction: true for long, false for short (default: true)")
	cmd.Flags().Float64Var(&openPrice, "open-price", 0, "Open price (optional: if not set, fetched from oracle and used as-is)")
	cmd.Flags().Uint64Var(&marketIndex, "market-index", 0, "Market index to trade")
	cmd.Flags().Uint64Var(&collateralIndex, "collateral-index", 0, "Collateral token index")
	cmd.Flags().StringVar(&tradeType, "trade-type", "", "Trade type: 'trade' (market), 'limit', or 'stop' (default: 'trade')")
	cmd.Flags().StringVar(&tradeJSONFile, "trade-json", "", "Path to JSON file with open_trade parameters (overrides dynamic trading)")

	return cmd
}

// newCloseCmd creates the close subcommand
func newCloseCmd() *cobra.Command {
	var tradeIndex uint64

	cmd := &cobra.Command{
		Use:   "close",
		Short: "Close a market trade order in Sai Perps",
		Long: `Close a market trade order (position) in Sai Perps.

This command sends a close_trade message to the perp contract to close a specific trade position.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClose(tradeIndex)
		},
	}

	cmd.Flags().Uint64Var(&tradeIndex, "trade-index", 0, "Trade index to close (UserTradeIndex)")
	cmd.MarkFlagRequired("trade-index")

	return cmd
}

// newListCmd creates the list subcommand
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available markets in Sai Perps",
		Long: `List all available markets in Sai Perps.

This command queries the perp contract to display all configured markets with their details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
		},
	}

	return cmd
}

// newPositionsCmd creates the positions subcommand
func newPositionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "positions",
		Short: "List all open positions/trades",
		Long: `List all open positions (trades) for the current account in Sai Perps.

This command queries the perp contract to display all trades/positions with their details.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return runPositions()
		},
	}

	return cmd
}

// newAutoCmd creates the auto subcommand
func newAutoCmd() *cobra.Command {
	var (
		configFile        string
		marketIndex       uint64
		collateralIndex   uint64
		minTradeSize      uint64
		maxTradeSize      uint64
		minLeverage       uint64
		maxLeverage       uint64
		blocksBeforeClose uint64
		maxOpenPositions  int
		loopDelaySeconds  int
	)

	cmd := &cobra.Command{
		Use:   "auto",
		Short: "Run automated random trading",
		Long: `Run an automated trading bot that randomly opens and closes positions.

This command continuously:
- Opens random trades with random parameters (size, leverage, direction)
- Tracks open positions and their opening block numbers
- Closes positions after a specified number of blocks
- Checks balance before opening trades to avoid insufficient funds

Examples:
  # Run with config file
  trader auto --config auto-trader.json

  # Run with defaults (market 0, random size/leverage, close after 20 blocks)
  trader auto

  # Custom parameters via flags (overrides config file)
  trader auto --config auto-trader.json --min-leverage 5 --max-leverage 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuto(configFile, marketIndex, collateralIndex, minTradeSize, maxTradeSize,
				minLeverage, maxLeverage, blocksBeforeClose, maxOpenPositions, loopDelaySeconds, cmd)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "Path to JSON config file (optional)")
	cmd.Flags().Uint64Var(&marketIndex, "market-index", 0, "Market index to trade")
	cmd.Flags().Uint64Var(&collateralIndex, "collateral-index", 0, "Collateral token index (default: use market's quote token)")
	cmd.Flags().Uint64Var(&minTradeSize, "min-trade-size", 1000000, "Minimum trade size in smallest units")
	cmd.Flags().Uint64Var(&maxTradeSize, "max-trade-size", 5000000, "Maximum trade size in smallest units")
	cmd.Flags().Uint64Var(&minLeverage, "min-leverage", 1, "Minimum leverage (e.g., 1 for 1x)")
	cmd.Flags().Uint64Var(&maxLeverage, "max-leverage", 10, "Maximum leverage (e.g., 10 for 10x)")
	cmd.Flags().Uint64Var(&blocksBeforeClose, "blocks-before-close", 20, "Number of blocks to wait before closing a position")
	cmd.Flags().IntVar(&maxOpenPositions, "max-open-positions", 5, "Maximum number of positions to keep open at once")
	cmd.Flags().IntVar(&loopDelaySeconds, "loop-delay", 5, "Delay in seconds between each loop iteration")

	return cmd
}

func runOpen(tradeSize, leverage uint64, long *bool, marketIndex, collateralIndex uint64, tradeType string, openPrice *float64, tradeJSONFile string) error {
	cfg, err := setupConfig(true)
	if err != nil {
		return err
	}

	// Strategy config
	cfg.TradeSize = tradeSize
	cfg.Leverage = leverage
	cfg.Long = long
	cfg.MarketIndex = marketIndex
	cfg.CollateralIndex = collateralIndex
	cfg.OpenPrice = openPrice
	// Validate trade-type if provided
	if tradeType != "" {
		if !evmtrader.IsValidTradeType(tradeType) {
			return fmt.Errorf("invalid trade-type: %s (must be '%s', '%s', or '%s')",
				tradeType, evmtrader.TradeTypeMarket, evmtrader.TradeTypeLimit, evmtrader.TradeTypeStop)
		}
		cfg.TradeType = tradeType
		cfg.EnableLimitOrder = evmtrader.IsLimitOrStopOrder(tradeType)
		// Note: --open-price is optional for limit/stop orders
		// If not provided, the price will be fetched from oracle and used as-is
	} else {
		// Default to market order if not specified
		cfg.TradeType = evmtrader.TradeTypeMarket
		cfg.EnableLimitOrder = false
	}
	cfg.TradeJSONFile = tradeJSONFile

	// Create trader
	ctx := context.Background()
	trader, err := createTrader(ctx, cfg)
	if err != nil {
		return err
	}
	defer trader.Close()

	// Open trade - use JSON file if provided, otherwise use config
	if cfg.TradeJSONFile != "" {
		if err := trader.OpenTradeFromJSON(ctx, cfg.TradeJSONFile); err != nil {
			return fmt.Errorf("open trade from JSON: %w", err)
		}
	} else {
		if err := trader.OpenTradeFromConfig(ctx); err != nil {
			return fmt.Errorf("open trade: %w", err)
		}
	}

	return nil
}

func runClose(tradeIndex uint64) error {
	cfg, err := setupConfig(true)
	if err != nil {
		return err
	}

	// Create trader
	ctx := context.Background()
	trader, err := createTrader(ctx, cfg)
	if err != nil {
		return err
	}
	defer trader.Close()

	// Close the trade
	if err := trader.CloseTrade(ctx, tradeIndex); err != nil {
		return fmt.Errorf("close trade: %w", err)
	}

	fmt.Printf("Successfully closed trade %d\n", tradeIndex)
	return nil
}

func runList() error {
	cfg, err := setupConfig(false)
	if err != nil {
		return err
	}

	// Create trader
	ctx := context.Background()
	trader, err := createTrader(ctx, cfg)
	if err != nil {
		return err
	}
	defer trader.Close()

	// Query markets
	markets, err := trader.QueryMarkets(ctx)
	if err != nil {
		return fmt.Errorf("query markets: %w", err)
	}

	// Display markets
	if len(markets) == 0 {
		fmt.Println("No markets found")
	} else {
		fmt.Println("Available Markets:")
		fmt.Println("==================")
		for _, market := range markets {
			fmt.Printf("Market Index: %d\n", market.Index)
			if market.BaseToken != nil {
				fmt.Printf("  Base Token: %d\n", *market.BaseToken)
			}
			if market.QuoteToken != nil {
				fmt.Printf("  Quote Token: %d\n", *market.QuoteToken)
			}
			if market.MaxOI != nil {
				fmt.Printf("  Max OI: %s\n", *market.MaxOI)
			}
			if market.FeePerBlock != nil {
				fmt.Printf("  Fee Per Block: %s\n", *market.FeePerBlock)
			}
			fmt.Println()
		}
	}

	// Query collaterals
	collaterals, err := trader.QueryCollaterals(ctx)
	if err != nil {
		// Don't fail if collaterals query fails, just log
		fmt.Fprintf(os.Stderr, "Warning: Failed to query collaterals: %v\n", err)
	} else if len(collaterals) > 0 {
		fmt.Println("Available Collaterals:")
		fmt.Println("======================")
		for _, collateral := range collaterals {
			fmt.Printf("Collateral Index: %d\n", collateral.Index)
			fmt.Printf("  Denom: %s\n", collateral.Denom)
			fmt.Println()
		}
	}

	return nil
}

func runPositions() error {
	cfg, err := setupConfig(true)
	if err != nil {
		return err
	}

	// Create trader
	ctx := context.Background()
	trader, err := createTrader(ctx, cfg)
	if err != nil {
		return fmt.Errorf("trader error: %w", err)
	}
	defer trader.Close()

	// Query and display positions
	if err := trader.QueryAndDisplayPositions(ctx); err != nil {
		return fmt.Errorf("trader error: %w", err)
	}

	return nil
}

func runAuto(configFile string, marketIndex, collateralIndex, minTradeSize, maxTradeSize, minLeverage, maxLeverage,
	blocksBeforeClose uint64, maxOpenPositions, loopDelaySeconds int, cmd *cobra.Command) error {

	var autoCfg evmtrader.AutoTradingConfig

	// Load from config file if provided
	if configFile != "" {
		jsonCfg, err := evmtrader.LoadAutoTradingConfig(configFile)
		if err != nil {
			return fmt.Errorf("load config file: %w", err)
		}

		// Convert JSON config to AutoTradingConfig
		autoCfg = jsonCfg.ToAutoTradingConfig()

		// Apply network settings from config if present
		if jsonCfg.Network != nil {
			if jsonCfg.Network.Mode != "" && !cmd.Flags().Changed("network") {
				networkMode = jsonCfg.Network.Mode
			}
			if jsonCfg.Network.EVMRPCUrl != "" && !cmd.Flags().Changed("evm-rpc") {
				evmRPCUrl = jsonCfg.Network.EVMRPCUrl
			}
			if jsonCfg.Network.NetworksToml != "" && !cmd.Flags().Changed("networks-toml") {
				networksTomlFile = jsonCfg.Network.NetworksToml
			}
		}

		fmt.Printf("Loaded config from: %s\n", configFile)
	} else {
		// Use command-line flags as defaults
		autoCfg = evmtrader.AutoTradingConfig{
			MarketIndex:       marketIndex,
			CollateralIndex:   collateralIndex,
			MinTradeSize:      minTradeSize,
			MaxTradeSize:      maxTradeSize,
			MinLeverage:       minLeverage,
			MaxLeverage:       maxLeverage,
			BlocksBeforeClose: blocksBeforeClose,
			MaxOpenPositions:  maxOpenPositions,
			LoopDelaySeconds:  loopDelaySeconds,
		}
	}

	// Override config file with command-line flags if they were explicitly set
	if cmd.Flags().Changed("market-index") {
		autoCfg.MarketIndex = marketIndex
	}
	if cmd.Flags().Changed("collateral-index") {
		autoCfg.CollateralIndex = collateralIndex
	}
	if cmd.Flags().Changed("min-trade-size") {
		autoCfg.MinTradeSize = minTradeSize
	}
	if cmd.Flags().Changed("max-trade-size") {
		autoCfg.MaxTradeSize = maxTradeSize
	}
	if cmd.Flags().Changed("min-leverage") {
		autoCfg.MinLeverage = minLeverage
	}
	if cmd.Flags().Changed("max-leverage") {
		autoCfg.MaxLeverage = maxLeverage
	}
	if cmd.Flags().Changed("blocks-before-close") {
		autoCfg.BlocksBeforeClose = blocksBeforeClose
	}
	if cmd.Flags().Changed("max-open-positions") {
		autoCfg.MaxOpenPositions = maxOpenPositions
	}
	if cmd.Flags().Changed("loop-delay") {
		autoCfg.LoopDelaySeconds = loopDelaySeconds
	}

	cfg, err := setupConfig(true)
	if err != nil {
		return err
	}

	// Create trader
	ctx := context.Background()
	trader, err := createTrader(ctx, cfg)
	if err != nil {
		return err
	}
	defer trader.Close()

	// Run the auto-trading loop
	if err := trader.RunAutoTrading(ctx, autoCfg); err != nil {
		return fmt.Errorf("auto trading: %w", err)
	}

	return nil
}

// setupConfig creates and configures an EVMTrader config with network settings and authentication.
// requireAuth determines whether a valid private key is required (false for read-only queries).
func setupConfig(requireAuth bool) (evmtrader.Config, error) {
	// Load .env file if it exists (best effort - ignore errors)
	_ = godotenv.Load(".env")

	cfg := evmtrader.Config{}

	// Try to load from TOML file first (unless --contracts-env is explicitly set for legacy mode)
	var grpcUrl, chainID string
	useTOML := contractsEnvFile == "" && networksTomlFile != ""

	if useTOML {
		// Check if TOML file exists
		if _, err := os.Stat(networksTomlFile); err == nil {
			// Load network config from TOML
			networkConfig, err := evmtrader.LoadNetworkConfig(networksTomlFile)
			if err != nil {
				// Fall back to hardcoded defaults if TOML fails to load
				fmt.Fprintf(os.Stderr, "Warning: Failed to load TOML config: %v, using hardcoded defaults\n", err)
				useTOML = false
			} else {
				netInfo, err := evmtrader.GetNetworkInfo(networkConfig, networkMode)
				if err != nil {
					return cfg, err
				}

				// Use TOML config unless overridden by flags
				if evmRPCUrl == "" {
					evmRPCUrl = netInfo.EVMRPCUrl
				}
				grpcUrl = netInfo.GrpcUrl
				chainID = netInfo.ChainID

				// Load contract addresses from TOML
				contractAddrs := evmtrader.ContractAddressesFromNetworkInfo(netInfo)
				cfg.ContractAddresses = &contractAddrs
			}
		} else {
			// TOML file doesn't exist, fall back to hardcoded defaults
			useTOML = false
		}
	}

	// Fall back to hardcoded defaults if not using TOML or if TOML failed
	if !useTOML {
		if evmRPCUrl == "" {
			switch networkMode {
			case "localnet":
				evmRPCUrl = "http://localhost:8545"
				grpcUrl = "localhost:9090"
				chainID = "nibiru-localnet-0"
			case "testnet":
				evmRPCUrl = "https://evm-rpc.testnet-2.nibiru.fi"
				grpcUrl = "grpc.testnet-2.nibiru.fi:443"
				chainID = "nibiru-testnet-2"
			case "mainnet":
				evmRPCUrl = "https://evm-rpc.nibiru.fi"
				grpcUrl = "grpc.nibiru.fi:443"
				chainID = "nibiru-mainnet-1"
			default:
				return cfg, fmt.Errorf("unknown network mode: %s (use: localnet, testnet, mainnet)", networkMode)
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
	}

	cfg.EVMRPCUrl = evmRPCUrl
	cfg.GrpcUrl = grpcUrl
	cfg.ChainID = chainID

	// Auto-detect contracts env file if not provided (legacy mode)
	if contractsEnvFile == "" {
		contractsEnvFile = detectContractsEnvFile(networkMode)
	}
	cfg.ContractsEnvFile = contractsEnvFile

	// Get private key or mnemonic: try flags first, then env
	privKey := privateKeyHex
	if privKey == "" {
		privKey = os.Getenv("EVM_PRIVATE_KEY")
	}

	mnem := mnemonic
	if mnem == "" {
		mnem = os.Getenv("EVM_MNEMONIC")
	}

	// If we have a mnemonic, convert it to private key using Nibiru's built-in EVM HD derivation
	if mnem != "" {
		privKeyHex, err := mnemonicToPrivateKeyHex(mnem)
		if err != nil {
			return cfg, fmt.Errorf("failed to convert mnemonic to private key: %w", err)
		}
		privKey = privKeyHex
	}

	// For queries that don't require signing, use a dummy key if none is provided
	if privKey == "" && !requireAuth {
		// Generate a dummy key just for query purposes (won't be used for signing)
		privKey = "0000000000000000000000000000000000000000000000000000000000000001"
	}

	if privKey == "" && requireAuth {
		return cfg, fmt.Errorf("private key or mnemonic required: set --private-key or --mnemonic flag, EVM_PRIVATE_KEY or EVM_MNEMONIC env var")
	}

	cfg.PrivateKeyHex = privKey

	// Load Slack webhook from environment
	cfg.SlackWebhook = os.Getenv("SLACK_WEBHOOK")

	// Load Slack error keywords filter from environment (comma-separated)
	keywordsEnv := os.Getenv("SLACK_ERROR_KEYWORDS")
	if keywordsEnv != "" {
		keywords := strings.Split(keywordsEnv, ",")
		for i := range keywords {
			keywords[i] = strings.TrimSpace(keywords[i])
		}
		cfg.SlackErrorKeywords = keywords
	}

	return cfg, nil
}

// createTrader creates and initializes a new EVMTrader with the given config.
// The caller is responsible for calling Close() on the returned trader.
func createTrader(ctx context.Context, cfg evmtrader.Config) (*evmtrader.EVMTrader, error) {
	trader, err := evmtrader.New(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create trader: %w", err)
	}
	return trader, nil
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

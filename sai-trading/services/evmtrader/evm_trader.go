package evmtrader

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"fmt"
	"math/big"
	"strings"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/eth/crypto/ethsecp256k1"
	"github.com/cosmos/cosmos-sdk/client"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// EVMTrader encapsulates the EVM client and trading routine.
type EVMTrader struct {
	cfg        Config
	client     *ethclient.Client
	txClient   txtypes.ServiceClient
	encCfg     client.TxConfig
	grpcConn   *grpc.ClientConn // Store gRPC connection for cleanup
	privKey    *ecdsa.PrivateKey
	ethPrivKey *ethsecp256k1.PrivKey

	accountAddr   common.Address
	ethAddrBech32 string

	cosmosAddr    string
	cosmosAddrHex common.Address

	addrs ContractAddresses
}

// New returns a new EVMTrader after validating configuration.
func New(ctx context.Context, cfg Config) (*EVMTrader, error) {
	// Normalize paths and set defaults
	normalizeConfigPaths(&cfg)
	setConfigDefaults(&cfg)

	if cfg.EVMRPCUrl == "" {
		return nil, fmt.Errorf("EVMRPCUrl is required")
	}
	if cfg.PrivateKeyHex == "" {
		return nil, fmt.Errorf("PrivateKeyHex is required")
	}

	// Connect to EVM RPC for queries
	client, err := ethclient.DialContext(ctx, cfg.EVMRPCUrl)
	if err != nil {
		return nil, fmt.Errorf("dial evm rpc: %w", err)
	}

	// Parse private key
	priv, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	accountAddr := crypto.PubkeyToAddress(priv.PublicKey)

	// Convert to ethsecp256k1 for keyring signer
	ethPrivKey := &ethsecp256k1.PrivKey{
		Key: crypto.FromECDSA(priv),
	}

	cosmosAddr := cfg.CosmosAddress
	var cosmosAddrHex common.Address
	var ethAddrBech32 string

	if cfg.Mnemonic != "" {
		accounts, err := DeriveAccountsFromMnemonic(cfg.Mnemonic, "nibi")
		if err != nil {
			return nil, fmt.Errorf("derive all addresses: %w", err)
		}

		cosmosAddr = accounts.CosmosAddrBech32
		cosmosAddrHex = accounts.CosmosAddrHex
		ethAddrBech32 = accounts.EthAddrBech32
	}

	// Connect to gRPC for transaction broadcasting
	// Use TLS for remote servers (testnet/mainnet), insecure for localhost
	var grpcCreds credentials.TransportCredentials
	if strings.Contains(cfg.GrpcUrl, ":443") || (!strings.Contains(cfg.GrpcUrl, "localhost") && !strings.Contains(cfg.GrpcUrl, "127.0.0.1")) {
		grpcCreds = credentials.NewTLS(&tls.Config{})
	} else {
		grpcCreds = insecure.NewCredentials()
	}
	grpcConn, err := grpc.DialContext(ctx, cfg.GrpcUrl,
		grpc.WithTransportCredentials(grpcCreds),
	)
	if err != nil {
		return nil, fmt.Errorf("dial grpc: %w", err)
	}

	// Get encoding config and tx client for direct EVM tx sending
	encCfg := getEncConfig()
	txClient := txtypes.NewServiceClient(grpcConn)

	// Load contract addresses: use from Config if provided, otherwise load from file
	var addrs ContractAddresses
	if cfg.ContractAddresses != nil {
		addrs = *cfg.ContractAddresses
	} else {
		var err error
		addrs, err = loadContractAddresses(cfg.ContractsEnvFile)
		if err != nil {
			return nil, fmt.Errorf("load contracts: %w", err)
		}
	}
	trader := &EVMTrader{
		cfg:        cfg,
		client:     client,
		txClient:   txClient,
		encCfg:     encCfg.TxConfig,
		grpcConn:   grpcConn,
		privKey:    priv,
		ethPrivKey: ethPrivKey,
		// Ethereum path (m/44'/60'/0'/0/0) - MetaMask - USED FOR TRADING
		accountAddr:   accountAddr,   // 0x1234... (hex, shown in MetaMask)
		ethAddrBech32: ethAddrBech32, // nibi1xyz... (bech32)
		// Cosmos path (m/44'/118'/0'/0/0) - Keplr
		cosmosAddr:    cosmosAddr,    // nibi1abc... (bech32, shown in Keplr)
		cosmosAddrHex: cosmosAddrHex, // 0xABC... (hex)
		addrs:         addrs,
	}

	return trader, nil
}

// Close releases underlying resources.
func (t *EVMTrader) Close() {
	if t.client != nil {
		t.client.Close()
	}

	if t.grpcConn != nil {
		if err := t.grpcConn.Close(); err != nil {
			t.logWarn("Failed to close gRPC connection", "error", err.Error())
		}
	}
}

// OpenTradeFromConfig opens a trade using the trader's configuration
func (t *EVMTrader) OpenTradeFromConfig(ctx context.Context) error {
	chainID, err := t.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("chain id: %w", err)
	}

	// Query Cosmos bank balance
	stNIBIDenom := t.addrs.StNIBIDenom
	if stNIBIDenom == "" {
		return fmt.Errorf("stNIBI denom not configured")
	}
	bal, err := t.queryCosmosBalance(ctx, t.ethAddrBech32, stNIBIDenom)
	if err != nil {
		return fmt.Errorf("query Cosmos balance: %w", err)
	}

	// Prepare trade from config
	params, err := t.prepareTradeFromConfig(ctx, bal)
	if err != nil {
		return fmt.Errorf("prepare trade: %w", err)
	}
	if params == nil {
		return nil // Insufficient balance or other skip condition
	}

	// Execute the trade
	return t.OpenTrade(ctx, chainID, params)
}

// OpenTrade opens a trade with the given parameters
func (t *EVMTrader) OpenTrade(ctx context.Context, chainID *big.Int, params *OpenTradeParams) error {
	// Validate that the market has pair_depths configured (required for price impact calculations)
	hasPairDepth, err := t.queryPairDepth(ctx, params.MarketIndex)
	if err != nil {
		return fmt.Errorf("check market configuration: %w", err)
	}
	if !hasPairDepth {
		return fmt.Errorf("market %d is not fully configured: missing pair_depths", params.MarketIndex)
	}

	// Build open_trade message
	msgBytes, err := t.buildOpenTradeMessage(params)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	// Send transaction
	txResp, err := t.sendOpenTradeTransaction(ctx, chainID, msgBytes, params.CollateralAmt, params.CollateralIndex)
	if err != nil {
		return fmt.Errorf("send transaction: %w", err)
	}

	// Parse trade ID from response
	isLimitOrder := isLimitOrStopOrder(params.TradeType)
	tradeID, err := t.parseTradeID(txResp)
	if err != nil {
		t.logError("Failed to parse trade ID", "error", err.Error(), "tx_hash", txResp.TxHash)
		return err
	}

	whatTraderOpens := "position"
	if isLimitOrder {
		whatTraderOpens = "limit order"
	}

	t.logInfo("Successfully opened trade",
		"type", whatTraderOpens,
		"trade_id", tradeID,
		"tx_hash", txResp.TxHash,
		"height", txResp.Height,
	)

	return nil
}

// CloseTrade closes a market trade order with the given trade index
func (t *EVMTrader) CloseTrade(ctx context.Context, tradeIndex uint64) error {
	chainID, err := t.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("chain id: %w", err)
	}

	// Build close_trade message
	msgBytes, err := t.buildCloseTradeMessage(tradeIndex)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	// Send transaction
	txResp, err := t.sendCloseTradeTransaction(ctx, chainID, msgBytes)
	if err != nil {
		return fmt.Errorf("send transaction: %w", err)
	}

	t.logInfo("Successfully closed trade",
		"trade_index", tradeIndex,
		"tx_hash", txResp.TxHash,
		"height", txResp.Height,
	)

	return nil
}

// getEncConfig returns the encoding configuration for the Nibiru chain.
func getEncConfig() app.EncodingConfig {
	return app.MakeEncodingConfig()
}

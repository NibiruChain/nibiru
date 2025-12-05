package evmtrader

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

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
	cfg         Config
	client      *ethclient.Client
	txClient    txtypes.ServiceClient
	encCfg      client.TxConfig
	grpcConn    *grpc.ClientConn // Store gRPC connection for cleanup
	privKey     *ecdsa.PrivateKey
	ethPrivKey  *ethsecp256k1.PrivKey
	accountAddr common.Address
	addrs       ContractAddresses
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
	return &EVMTrader{
		cfg:         cfg,
		client:      client,
		txClient:    txClient,
		encCfg:      encCfg.TxConfig,
		grpcConn:    grpcConn,
		privKey:     priv,
		ethPrivKey:  ethPrivKey,
		accountAddr: accountAddr,
		addrs:       addrs,
	}, nil
}

// Close releases underlying resources.
func (t *EVMTrader) Close() {
	if t.client != nil {
		t.client.Close()
	}

	if t.grpcConn != nil {
		if err := t.grpcConn.Close(); err != nil {
			t.log("Failed to close gRPC connection", "error", err.Error())
		}
	}
}

// OpenTradeFromConfig opens a trade using the trader's configuration
func (t *EVMTrader) OpenTradeFromConfig(ctx context.Context) error {
	chainID, err := t.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("chain id: %w", err)
	}

	// Query ERC20 balance
	erc20ABI := getERC20ABI()
	erc20Addr := common.HexToAddress(t.addrs.TokenStNIBIERC20)
	bal, err := t.queryERC20Balance(ctx, erc20ABI, erc20Addr, t.accountAddr)
	if err != nil {
		return fmt.Errorf("query ERC20 balance: %w", err)
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
		t.log("Failed to parse trade ID", "error", err.Error(), "tx_hash", txResp.TxHash)
		return err
	}

	whatTraderOpens := "position"
	if isLimitOrder {
		whatTraderOpens = "limit order"
	}

	t.log("Successfully opened trade",
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

	t.log("Successfully closed trade",
		"trade_index", tradeIndex,
		"tx_hash", txResp.TxHash,
		"height", txResp.Height,
	)

	return nil
}

// log is a minimal structured logger.
func (t *EVMTrader) log(msg string, kv ...any) {
	fields := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		k, _ := kv[i].(string)
		fields[k] = kv[i+1]
	}
	fields["msg"] = msg
	fields["ts"] = time.Now().UTC().Format(time.RFC3339)
	_ = json.NewEncoder(os.Stdout).Encode(fields)
}

// logError logs an error and optionally sends it to Slack webhook
func (t *EVMTrader) logError(msg string, kv ...any) {
	t.log(msg, kv...)

	// Check if Slack webhook is configured
	slackWebhook := os.Getenv("SLACK_WEBHOOK")
	if slackWebhook == "" {
		return
	}

	// Build error message for Slack
	errorFields := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		k, _ := kv[i].(string)
		errorFields[k] = kv[i+1]
	}

	// Format Slack message
	slackMsg := map[string]interface{}{
		"text": fmt.Sprintf("🚨 Auto-Trader Error: %s", msg),
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s*\n\n*Details:*", msg),
				},
			},
			{
				"type":   "section",
				"fields": buildSlackFields(errorFields),
			},
		},
	}

	// Send to Slack (non-blocking)
	go sendSlackNotification(slackWebhook, slackMsg)
}

// buildSlackFields converts error fields to Slack field format
func buildSlackFields(fields map[string]any) []map[string]interface{} {
	slackFields := []map[string]interface{}{}
	for k, v := range fields {
		slackFields = append(slackFields, map[string]interface{}{
			"type": "mrkdwn",
			"text": fmt.Sprintf("*%s:*\n%s", k, fmt.Sprintf("%v", v)),
		})
		if len(slackFields) >= 10 { // Slack has a limit on fields
			break
		}
	}
	return slackFields
}

// sendSlackNotification sends a notification to Slack webhook
func sendSlackNotification(webhookURL string, payload map[string]interface{}) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return // Silently fail if we can't marshal
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return // Silently fail if we can't create request
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return // Silently fail if request fails
	}
	defer resp.Body.Close()
}

// getEncConfig returns the encoding configuration for the Nibiru chain.
func getEncConfig() app.EncodingConfig {
	return app.MakeEncodingConfig()
}

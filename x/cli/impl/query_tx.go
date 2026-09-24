package impl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
	evmrpc "github.com/NibiruChain/nibiru/v2/evm/rpc"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/query"
	authcmd "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/client/cli"
	authtx "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/tx"
)

const (
	queryTxTypeFlag   = "type"
	queryTxTypeHash   = "hash"
	queryTxTypeAccSeq = "acc_seq"
	queryTxTypeSig    = "signature"
	flagEVMView       = "evm"
	flagDetails       = "details"
	flagEVMRPC        = "evm-rpc"
)

var publicEVMRPCEndpoints = map[string]string{
	appconst.SDK_CHAIN_ID_MAINNET: "https://evm-rpc.nibiru.fi",
	"nibiru-testnet-2":            "https://evm-rpc.testnet-2.nibiru.fi",
}

// TransactionSummary preserves the native TxResponse fields for committed
// transactions and adds message types and EVM transaction information.
type TransactionSummary struct {
	TxHash    string                 `json:"txhash"`
	EthHash   string                 `json:"eth_hash"`
	Height    *string                `json:"height,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
	Code      *uint32                `json:"code,omitempty"`
	Codespace string                 `json:"codespace,omitempty"`
	Data      string                 `json:"data"`
	Info      string                 `json:"info"`
	GasWanted *string                `json:"gas_wanted,omitempty"`
	GasUsed   *string                `json:"gas_used,omitempty"`
	Messages  []string               `json:"messages,omitempty"`
	RawLog    string                 `json:"raw_log"`
	Logs      json.RawMessage        `json:"logs"`
	Events    json.RawMessage        `json:"events"`
	Tx        json.RawMessage        `json:"tx,omitempty"`
	EVM       *EVMTransactionSummary `json:"evm,omitempty"`
	Pending   bool                   `json:"pending,omitempty"`
}

// EVMTransactionSummary contains the EVM fields users commonly inspect while
// retaining the smaller default transaction response.
type EVMTransactionSummary struct {
	Hash             gethcommon.Hash     `json:"hash"`
	BlockHash        *gethcommon.Hash    `json:"block_hash,omitempty"`
	BlockNumber      *hexutil.Big        `json:"block_number,omitempty"`
	From             gethcommon.Address  `json:"from"`
	Gas              hexutil.Uint64      `json:"gas"`
	GasPrice         *hexutil.Big        `json:"gas_price"`
	MaxFeePerGas     *hexutil.Big        `json:"max_fee_per_gas,omitempty"`
	MaxPriorityFee   *hexutil.Big        `json:"max_priority_fee_per_gas,omitempty"`
	Input            hexutil.Bytes       `json:"input"`
	Nonce            hexutil.Uint64      `json:"nonce"`
	To               *gethcommon.Address `json:"to"`
	TransactionIndex *hexutil.Uint64     `json:"transaction_index,omitempty"`
	Value            *hexutil.Big        `json:"value"`
	Type             hexutil.Uint64      `json:"type"`
}

type transactionDetails struct {
	TransactionSummary
	Comet  json.RawMessage      `json:"comet,omitempty"`
	EVMRPC *evmrpc.EthTxJsonRPC `json:"evm_rpc,omitempty"`
}

type ethereumJSONRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// QueryTxCmd returns the Nibiru-owned q tx command. Hash lookups support both
// CometBFT and EVM hashes. The account-sequence and signature query forms keep
// the Cosmos SDK behavior.
func QueryTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx [hash]",
		Short: "Query a transaction by CometBFT hash or EVM transaction hash",
		Long: strings.TrimSpace(`Query a transaction by a bare CometBFT hash or a 0x-prefixed EVM hash.

By default, hash lookups print the native Comet transaction fields with eth_hash and message types.
Use --details for the complete Comet transaction under comet and the native EVM transaction under evm_rpc when available.
Use --evm with an EVM hash to print the native eth_getTransactionByHash JSON response.`),
		Example: strings.TrimSpace(`
nibid q tx 624CAA0738E95CE04C22D4D012FF67AD86521FB93B2F64CE9967A8FE31EAB3A1
nibid q tx 0x66ef47cf9bda9bfb9c5e465a44e9a825bf1d8f013ed9b4abaf6960c02e9d85fc
nibid q tx --details 0x66ef47cf9bda9bfb9c5e465a44e9a825bf1d8f013ed9b4abaf6960c02e9d85fc
nibid q tx --evm 0x66ef47cf9bda9bfb9c5e465a44e9a825bf1d8f013ed9b4abaf6960c02e9d85fc
`),
		Args: cobra.ExactArgs(1),
		RunE: runQueryTxCmd,
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().String(queryTxTypeFlag, queryTxTypeHash, "The type to use when querying tx: hash, acc_seq, or signature")
	cmd.Flags().Bool(flagEVMView, false, "Print the native eth_getTransactionByHash JSON response; requires a 0x-prefixed EVM hash")
	cmd.Flags().Bool(flagDetails, false, "Include the complete Comet transaction response and native EVM transaction when available")
	cmd.Flags().String(flagEVMRPC, "", "EVM JSON-RPC endpoint for --evm and pending EVM transaction lookup")

	return cmd
}

func runQueryTxCmd(cmd *cobra.Command, args []string) error {
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}

	typ, err := cmd.Flags().GetString(queryTxTypeFlag)
	if err != nil {
		return err
	}
	if typ != queryTxTypeHash {
		return runSDKTransactionQuery(cmd, clientCtx, typ, args[0])
	}

	showEVM, err := cmd.Flags().GetBool(flagEVMView)
	if err != nil {
		return err
	}
	showDetails, err := cmd.Flags().GetBool(flagDetails)
	if err != nil {
		return err
	}
	if showEVM && showDetails {
		return fmt.Errorf("--%s and --%s cannot be used together", flagEVMView, flagDetails)
	}

	hash, err := ClassifyTxHash(args[0])
	if err != nil {
		return err
	}
	if showEVM {
		evmHash, ok := hash.(EVMTxHash)
		if !ok {
			return fmt.Errorf("--%s requires a 0x-prefixed EVM transaction hash", flagEVMView)
		}
		endpoint, err := evmRPCEndpoint(cmd, clientCtx)
		if err != nil {
			return err
		}
		raw, _, err := queryEthereumTransaction(cmd.Context(), endpoint, evmHash.Hash())
		if err != nil {
			return err
		}
		return printJSON(cmd, json.RawMessage(raw))
	}

	var resolved resolvedTransaction
	switch typedHash := hash.(type) {
	case CometBFTTxHash:
		resolved, err = resolveCometTransaction(cmd.Context(), clientCtx, typedHash)
	case EVMTxHash:
		resolved, err = resolveEVMTransaction(cmd.Context(), cmd, clientCtx, typedHash)
	default:
		return fmt.Errorf("unsupported transaction hash type %T", hash)
	}
	if err != nil {
		return err
	}

	if !showDetails {
		return printJSON(cmd, resolved.summary)
	}

	var cometJSON json.RawMessage
	if resolved.cosmos != nil {
		cometJSON, err = clientCtx.Codec.MarshalJSON(resolved.cosmos)
		if err != nil {
			return err
		}
	}
	return printJSON(cmd, transactionDetails{
		TransactionSummary: resolved.summary,
		Comet:              cometJSON,
		EVMRPC:             resolved.evm,
	})
}

func runSDKTransactionQuery(cmd *cobra.Command, clientCtx client.Context, typ, input string) error {
	showEVM, _ := cmd.Flags().GetBool(flagEVMView)
	showDetails, _ := cmd.Flags().GetBool(flagDetails)
	if showEVM || showDetails {
		return fmt.Errorf("--%s and --%s require --%s=%s", flagEVMView, flagDetails, queryTxTypeFlag, queryTxTypeHash)
	}

	switch typ {
	case queryTxTypeSig:
		sigParts, err := authcmd.ParseSigArgs([]string{input})
		if err != nil {
			return err
		}
		events := make([]string, len(sigParts))
		for i, sig := range sigParts {
			events[i] = fmt.Sprintf("%s.%s='%s'", sdk.EventTypeTx, sdk.AttributeKeySignature, sig)
		}
		txs, err := authtx.QueryTxsByEvents(clientCtx, events, query.DefaultPage, query.DefaultLimit, "")
		if err != nil {
			return err
		}
		if len(txs.Txs) == 0 {
			return fmt.Errorf("found no txs matching given signatures")
		}
		if len(txs.Txs) > 1 {
			return fmt.Errorf("found %d txs matching given signatures", len(txs.Txs))
		}
		return clientCtx.PrintProto(txs.Txs[0])
	case queryTxTypeAccSeq:
		if input == "" {
			return fmt.Errorf("`acc_seq` type takes an argument '<addr>/<seq>'")
		}
		events := []string{fmt.Sprintf("%s.%s='%s'", sdk.EventTypeTx, sdk.AttributeKeyAccountSequence, input)}
		txs, err := authtx.QueryTxsByEvents(clientCtx, events, query.DefaultPage, query.DefaultLimit, "")
		if err != nil {
			return err
		}
		if len(txs.Txs) == 0 {
			return fmt.Errorf("found no txs matching given address and sequence combination")
		}
		if len(txs.Txs) > 1 {
			return fmt.Errorf("found %d txs matching given address and sequence combination", len(txs.Txs))
		}
		return clientCtx.PrintProto(txs.Txs[0])
	default:
		return fmt.Errorf("unknown --%s value %s", queryTxTypeFlag, typ)
	}
}

type resolvedTransaction struct {
	summary TransactionSummary
	cosmos  *sdk.TxResponse
	evm     *evmrpc.EthTxJsonRPC
}

func resolveCometTransaction(ctx context.Context, clientCtx client.Context, hash CometBFTTxHash) (resolvedTransaction, error) {
	node, err := clientCtx.GetNode()
	if err != nil {
		return resolvedTransaction{}, err
	}
	result, err := node.Tx(ctx, hash.Bytes(), true)
	if err != nil {
		return resolvedTransaction{}, err
	}
	cosmos, err := authtx.QueryTx(clientCtx, hash.Canonical())
	if err != nil {
		return resolvedTransaction{}, err
	}
	if cosmos.Empty() {
		return resolvedTransaction{}, fmt.Errorf("no transaction found with hash %s", hash.Canonical())
	}
	summary, err := summaryFromResult(clientCtx, result, cosmos, nil)
	if err != nil {
		return resolvedTransaction{}, err
	}
	return resolvedTransaction{summary: summary, cosmos: cosmos}, nil
}

func resolveEVMTransaction(ctx context.Context, cmd *cobra.Command, clientCtx client.Context, hash EVMTxHash) (resolvedTransaction, error) {
	result, err := findCommittedEVMTransaction(ctx, clientCtx, hash.Hash())
	if err != nil {
		return resolvedTransaction{}, err
	}
	if result == nil {
		endpoint, endpointErr := evmRPCEndpoint(cmd, clientCtx)
		if endpointErr != nil {
			return resolvedTransaction{}, endpointErr
		}
		_, pendingTx, pendingErr := queryEthereumTransaction(ctx, endpoint, hash.Hash())
		if pendingErr != nil {
			return resolvedTransaction{}, pendingErr
		}
		return resolvedTransaction{summary: evmRPCTransactionSummary(pendingTx), evm: pendingTx}, nil
	}

	cometHash := eth.TmTxHashToString(result.Hash)
	cosmos, err := authtx.QueryTx(clientCtx, cometHash)
	if err != nil {
		return resolvedTransaction{}, err
	}
	evmTx, err := evmTransactionFromCommittedResult(ctx, clientCtx, result, hash.Hash())
	if err != nil {
		return resolvedTransaction{}, err
	}
	summary, err := summaryFromResult(clientCtx, result, cosmos, evmTx)
	if err != nil {
		return resolvedTransaction{}, err
	}
	summary.EthHash = hash.Canonical()
	summary.EVM = summaryEVMTransaction(evmTx)
	return resolvedTransaction{summary: summary, cosmos: cosmos, evm: evmTx}, nil
}

func findCommittedEVMTransaction(ctx context.Context, clientCtx client.Context, hash gethcommon.Hash) (*coretypes.ResultTx, error) {
	node, err := clientCtx.GetNode()
	if err != nil {
		return nil, err
	}
	page, limit := 1, 2
	queryText := fmt.Sprintf("%s.%s='%s'", evm.PendingEthereumTxEvent, evm.PendingEthereumTxEventAttrEthHash, hash.Hex())
	results, err := node.TxSearch(ctx, queryText, true, &page, &limit, "")
	if err != nil {
		return nil, err
	}
	if len(results.Txs) == 0 {
		return nil, nil
	}
	if len(results.Txs) != 1 {
		return nil, fmt.Errorf("expected one committed transaction for EVM hash %s, found %d", hash.Hex(), len(results.Txs))
	}
	return results.Txs[0], nil
}

func evmTransactionFromCommittedResult(ctx context.Context, clientCtx client.Context, result *coretypes.ResultTx, wanted gethcommon.Hash) (*evmrpc.EthTxJsonRPC, error) {
	tx, err := clientCtx.TxConfig.TxDecoder()(result.Tx)
	if err != nil {
		return nil, err
	}
	var msg *evm.MsgEthereumTx
	for _, sdkMsg := range tx.GetMsgs() {
		candidate, ok := sdkMsg.(*evm.MsgEthereumTx)
		if ok && candidate.AsTransaction().Hash() == wanted {
			msg = candidate
			break
		}
	}
	if msg == nil {
		return nil, fmt.Errorf("committed transaction %s does not contain EVM hash %s", eth.TmTxHashToString(result.Hash), wanted.Hex())
	}

	evmIndex, err := evmTransactionIndex(result, wanted)
	if err != nil {
		return nil, err
	}
	node, err := clientCtx.GetNode()
	if err != nil {
		return nil, err
	}
	block, err := node.Block(ctx, &result.Height)
	if err != nil {
		return nil, err
	}
	if block.Block == nil {
		return nil, fmt.Errorf("block %d is empty", result.Height)
	}
	blockHash := gethcommon.BytesToHash(block.BlockID.Hash.Bytes())
	response := evmrpc.NewRPCTxFromMsgEthTx(
		msg,
		blockHash,
		uint64(result.Height),
		uint64(evmIndex),
		evm.WalletZeroBaseFeeWei(),
		eth.ParseEthChainID(clientCtx.ChainID),
	)
	response.GasPrice = (*hexutil.Big)(evm.WalletZeroBaseFeeWei())
	return response, nil
}

func evmTransactionIndex(result *coretypes.ResultTx, wanted gethcommon.Hash) (int32, error) {
	for _, event := range result.TxResult.Events {
		if event.Type != evm.PendingEthereumTxEvent {
			continue
		}
		hash, index, err := evm.GetEthHashAndIndexFromPendingEthereumTxEvent(event)
		if err != nil {
			return 0, err
		}
		if hash == wanted {
			return index, nil
		}
	}
	return 0, fmt.Errorf("committed transaction does not expose an EVM index for %s", wanted.Hex())
}

func summaryFromResult(clientCtx client.Context, result *coretypes.ResultTx, response *sdk.TxResponse, evmTx *evmrpc.EthTxJsonRPC) (TransactionSummary, error) {
	height := fmt.Sprint(response.Height)
	code := response.Code
	gasWanted := fmt.Sprint(response.GasWanted)
	gasUsed := fmt.Sprint(response.GasUsed)
	nativeJSON, err := clientCtx.Codec.MarshalJSON(response)
	if err != nil {
		return TransactionSummary{}, err
	}
	var nativeFields map[string]json.RawMessage
	if err := json.Unmarshal(nativeJSON, &nativeFields); err != nil {
		return TransactionSummary{}, err
	}
	messages, ethHash, err := transactionMessages(clientCtx, result.Tx)
	if err != nil {
		return TransactionSummary{}, err
	}
	summary := TransactionSummary{
		TxHash:    response.TxHash,
		EthHash:   ethHash,
		Height:    &height,
		Timestamp: response.Timestamp,
		Code:      &code,
		Codespace: response.Codespace,
		Data:      response.Data,
		Info:      response.Info,
		GasWanted: &gasWanted,
		GasUsed:   &gasUsed,
		Messages:  messages,
		RawLog:    response.RawLog,
		Logs:      nativeFields["logs"],
		Events:    nativeFields["events"],
		Tx:        nativeFields["tx"],
	}
	if evmTx != nil {
		summary.EVM = summaryEVMTransaction(evmTx)
	}
	return summary, nil
}

func transactionMessages(clientCtx client.Context, txBytes []byte) ([]string, string, error) {
	tx, err := clientCtx.TxConfig.TxDecoder()(txBytes)
	if err != nil {
		return nil, "", err
	}
	return describeTransactionMessages(tx.GetMsgs())
}

func describeTransactionMessages(txMsgs []sdk.Msg) ([]string, string, error) {
	messages := make([]string, 0, len(txMsgs))
	var ethHash string
	for _, msg := range txMsgs {
		messages = append(messages, sdk.MsgTypeURL(msg))
		if evmMsg, ok := msg.(*evm.MsgEthereumTx); ok && ethHash == "" {
			ethTx, err := evmMsg.AsTransactionSafe()
			if err != nil {
				return nil, "", err
			}
			ethHash = ethTx.Hash().Hex()
		}
	}
	return messages, ethHash, nil
}

func summaryEVMTransaction(tx *evmrpc.EthTxJsonRPC) *EVMTransactionSummary {
	if tx == nil {
		return nil
	}
	return &EVMTransactionSummary{
		Hash:             tx.Hash,
		BlockHash:        tx.BlockHash,
		BlockNumber:      tx.BlockNumber,
		From:             tx.From,
		Gas:              tx.Gas,
		GasPrice:         tx.GasPrice,
		MaxFeePerGas:     tx.GasFeeCap,
		MaxPriorityFee:   tx.GasTipCap,
		Input:            tx.Input,
		Nonce:            tx.Nonce,
		To:               tx.To,
		TransactionIndex: tx.TransactionIndex,
		Value:            tx.Value,
		Type:             tx.Type,
	}
}

// evmRPCTransactionSummary handles an EVM RPC result that has no matching
// CometBFT transaction-index record. A missing index record does not prove a
// transaction is pending, so Pending follows the EVM RPC block-hash rule.
func evmRPCTransactionSummary(tx *evmrpc.EthTxJsonRPC) TransactionSummary {
	return TransactionSummary{
		EthHash: tx.Hash.Hex(),
		Logs:    json.RawMessage(`[]`),
		Events:  json.RawMessage(`[]`),
		EVM:     summaryEVMTransaction(tx),
		Pending: tx.BlockHash == nil,
	}
}

func evmRPCEndpoint(cmd *cobra.Command, clientCtx client.Context) (string, error) {
	endpoint, err := cmd.Flags().GetString(flagEVMRPC)
	if err != nil {
		return "", err
	}
	if endpoint != "" {
		return endpoint, nil
	}
	endpoint, ok := publicEVMRPCEndpoints[clientCtx.ChainID]
	if !ok {
		return "", fmt.Errorf("no EVM JSON-RPC endpoint is configured for chain %q; provide --%s", clientCtx.ChainID, flagEVMRPC)
	}
	return endpoint, nil
}

func queryEthereumTransaction(ctx context.Context, endpoint string, hash gethcommon.Hash) ([]byte, *evmrpc.EthTxJsonRPC, error) {
	requestBody, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getTransactionByHash",
		"params":  []string{hash.Hex()},
	})
	if err != nil {
		return nil, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("EVM JSON-RPC request failed: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, fmt.Errorf("EVM JSON-RPC request returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	var rpcResponse ethereumJSONRPCResponse
	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		return nil, nil, err
	}
	if rpcResponse.Error != nil {
		return nil, nil, fmt.Errorf("EVM JSON-RPC error %d: %s", rpcResponse.Error.Code, rpcResponse.Error.Message)
	}
	if len(rpcResponse.Result) == 0 || string(rpcResponse.Result) == "null" {
		return nil, nil, fmt.Errorf("no EVM transaction found with hash %s", hash.Hex())
	}
	var transaction evmrpc.EthTxJsonRPC
	if err := json.Unmarshal(rpcResponse.Result, &transaction); err != nil {
		return nil, nil, fmt.Errorf("invalid eth_getTransactionByHash response: %w", err)
	}
	return rpcResponse.Result, &transaction, nil
}

func printJSON(cmd *cobra.Command, value any) error {
	output, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(output))
	return nil
}

package evmtrader

import (
	"context"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmtest "github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// sendEVMTransaction sends an EVM transaction using ChainClient and MsgEthereumTx
func (t *EVMTrader) sendEVMTransaction(ctx context.Context, to common.Address, value *big.Int, data []byte, chainID *big.Int) (*sdk.TxResponse, error) {
	// Get nonce from EVM client
	nonce, err := t.client.PendingNonceAt(ctx, t.accountAddr)
	if err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	// Estimate gas
	msg := ethereum.CallMsg{
		From: t.accountAddr,
		To:   &to,
		Gas:  0,
		Data: data,
	}
	gasLimit, err := t.client.EstimateGas(ctx, msg)
	if err != nil || gasLimit == 0 {
		gasLimit = 2_000_000
	}

	// Get gas price from network
	gasPrice, err := t.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("gas price: %w", err)
	}
	// Ensure minimum gas price (defensive: prevent 0 or extremely low values)
	if gasPrice.Cmp(big.NewInt(1000)) < 0 {
		gasPrice = big.NewInt(1000) // 1000 wei minimum
	}

	// Create JsonTxArgs
	txArgs := evm.JsonTxArgs{
		From:     &t.accountAddr,
		To:       &to,
		Nonce:    (*hexutil.Uint64)(&nonce),
		Gas:      (*hexutil.Uint64)(&gasLimit),
		GasPrice: (*hexutil.Big)(gasPrice),
		Value:    (*hexutil.Big)(value),
		Data:     (*hexutil.Bytes)(&data),
		ChainID:  (*hexutil.Big)(chainID),
	}

	// Convert to MsgEthereumTx
	ethTxMsg := txArgs.ToMsgEthTx()
	ethTxMsg.From = t.accountAddr.Hex()

	// Create signers
	gethSigner := ethtypes.LatestSignerForChainID(chainID)
	krSigner := evmtest.NewSigner(t.ethPrivKey)

	// Sign the transaction
	if err := ethTxMsg.Sign(gethSigner, krSigner); err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}

	// Build the Cosmos SDK transaction from the signed MsgEthereumTx
	// This handles the conversion properly and the tx is already signed
	txBuilder := t.encCfg.NewTxBuilder()
	cosmosTx, err := ethTxMsg.BuildTx(txBuilder, "unibi")
	if err != nil {
		return nil, fmt.Errorf("build tx: %w", err)
	}

	// Encode the transaction
	txBytes, err := t.encCfg.TxEncoder()(cosmosTx)
	if err != nil {
		return nil, fmt.Errorf("encode tx: %w", err)
	}

	// Broadcast via gRPC
	grpcRes, err := t.txClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
		TxBytes: txBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("broadcast tx: %w", err)
	}

	if grpcRes.TxResponse.Code != 0 {
		return nil, parseContractError(grpcRes.TxResponse.Code, grpcRes.TxResponse.RawLog)
	}

	// Wait for transaction to be committed
	txHash := grpcRes.TxResponse.TxHash
	timeout := time.NewTimer(15 * time.Second)
	tick := time.NewTicker(500 * time.Millisecond)
	defer timeout.Stop()
	defer tick.Stop()

	maxRetries := 30
	for retries := 0; retries < maxRetries; retries++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-tick.C:
			resp, _ := t.txClient.GetTx(ctx, &txtypes.GetTxRequest{Hash: txHash})
			if resp != nil && resp.TxResponse != nil {
				return resp.TxResponse, nil
			}
		case <-timeout.C:
			return nil, fmt.Errorf("tx not found after timeout, hash: %s", txHash)
		}
	}

	return nil, fmt.Errorf("tx not found after %d retries, hash: %s", maxRetries, txHash)
}

// sendOpenTradeTransaction sends the open_trade transaction
func (t *EVMTrader) sendOpenTradeTransaction(ctx context.Context, chainID *big.Int, msgBytes []byte, collateralAmt *big.Int, collateralIndex uint64) (*sdk.TxResponse, error) {
	// Build WASM execute call
	wasmABI := getWasmPrecompileABI()
	wasmPrecompileAddr := precompile.PrecompileAddr_Wasm

	// Query the correct denomination for the collateral index
	collateralDenom, err := t.queryCollateralDenom(ctx, collateralIndex)
	if err != nil {
		// Provide helpful error message suggesting common alternatives
		return nil, fmt.Errorf("query collateral denom for index %d: %w", collateralIndex, err)
	}

	funds := []struct {
		Denom  string
		Amount *big.Int
	}{
		{Denom: collateralDenom, Amount: collateralAmt},
	}

	data, err := wasmABI.Pack("execute", t.addrs.PerpAddress, msgBytes, funds)
	if err != nil {
		return nil, fmt.Errorf("pack wasm execute: %w", err)
	}

	// Sign and send EVM tx to WASM precompile
	return t.sendEVMTransaction(ctx, wasmPrecompileAddr, big.NewInt(0), data, chainID)
}

// sendCloseTradeTransaction sends the close_trade transaction
func (t *EVMTrader) sendCloseTradeTransaction(ctx context.Context, chainID *big.Int, msgBytes []byte) (*sdk.TxResponse, error) {
	// Build WASM execute call
	wasmABI := getWasmPrecompileABI()
	wasmPrecompileAddr := precompile.PrecompileAddr_Wasm

	// No funds needed for close_trade
	funds := []struct {
		Denom  string
		Amount *big.Int
	}{}

	data, err := wasmABI.Pack("execute", t.addrs.PerpAddress, msgBytes, funds)
	if err != nil {
		return nil, fmt.Errorf("pack wasm execute: %w", err)
	}

	// Sign and send EVM tx to WASM precompile
	return t.sendEVMTransaction(ctx, wasmPrecompileAddr, big.NewInt(0), data, chainID)
}

// parseContractError parses common contract errors and provides user-friendly error messages.
func parseContractError(code uint32, rawLog string) error {
	// Parse exposure limit error
	if strings.Contains(rawLog, "exposure limit reached") {
		// Extract OI values: "pair oi collateral: 1210358/1000000"
		re := regexp.MustCompile(`pair oi collateral: (\d+)/(\d+)`)
		matches := re.FindStringSubmatch(rawLog)
		if len(matches) == 3 {
			currentOI, _ := strconv.ParseUint(matches[1], 10, 64)
			maxOI, _ := strconv.ParseUint(matches[2], 10, 64)
			pctUsed := float64(currentOI) / float64(maxOI) * 100

			return fmt.Errorf(`❌ MARKET EXPOSURE LIMIT REACHED

Current Open Interest: %d
Maximum Allowed:       %d
Capacity Used:         %.1f%%

This market cannot accept new positions (long OR short) until some positions are closed.

Solutions:
  1. Try a different market:    trader list
  2. Wait for positions to close
  3. Check market status regularly

Error code: %d`, currentOI, maxOI, pctUsed, code)
		}
		return fmt.Errorf("market exposure limit reached - cannot open new positions (long or short)\n\nTry: trader list (to see other markets)\n\nError code: %d, log: %s", code, rawLog)
	}

	// Parse insufficient balance error
	if strings.Contains(rawLog, "insufficient funds") || strings.Contains(rawLog, "insufficient balance") {
		return fmt.Errorf("❌ INSUFFICIENT BALANCE\n\nYou don't have enough collateral tokens for this trade.\n\nCheck balance: trader positions\n\nError code: %d", code)
	}

	// Parse invalid market error
	if strings.Contains(rawLog, "market not found") || strings.Contains(rawLog, "invalid market") {
		return fmt.Errorf("❌ INVALID MARKET\n\nThe specified market doesn't exist.\n\nSee available markets: trader list\n\nError code: %d", code)
	}

	// Parse leverage error
	if strings.Contains(rawLog, "leverage") && (strings.Contains(rawLog, "too high") || strings.Contains(rawLog, "exceeds maximum")) {
		return fmt.Errorf("❌ LEVERAGE TOO HIGH\n\nThe requested leverage exceeds the market's maximum.\n\nError code: %d, log: %s", code, rawLog)
	}

	// Default error (show raw log)
	return fmt.Errorf("transaction failed (code=%d)\n\nContract error:\n%s\n\nTip: Run 'trader list' to check market status", code, rawLog)
}

// Copyright (c) 2023-2024 Nibi, Inc.
package types

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	tmtypes "github.com/cometbft/cometbft/types"

	errorsmod "cosmossdk.io/errors"
	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	evmtypes "github.com/NibiruChain/nibiru/x/evm/types"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethmath "github.com/ethereum/go-ethereum/common/math"
	gethcoretypes "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"
)

// ErrExceedBlockGasLimit defines the error message when tx execution exceeds the block gas limit.
// The tx fee is deducted in ante handler, so it shouldn't be ignored in JSON-RPC API.
const ErrExceedBlockGasLimit = "out of gas in location: block gas meter; gasWanted:"

// ErrStateDBCommit defines the error message when commit after executing EVM
// transaction, for example transfer native token to a distribution module
// account 0x93354845030274cD4bf1686Abd60AB28EC52e1a7 using an evm transaction.
// Note, the transfer amount cannot be set to 0, otherwise this problem will not
// be triggered.
const ErrStateDBCommit = "failed to commit stateDB"

// RawTxToEthTx returns a evm MsgEthereum transaction from raw tx bytes.
func RawTxToEthTx(clientCtx client.Context, txBz tmtypes.Tx) ([]*evmtypes.MsgEthereumTx, error) {
	tx, err := clientCtx.TxConfig.TxDecoder()(txBz)
	if err != nil {
		return nil, errorsmod.Wrap(errortypes.ErrJSONUnmarshal, err.Error())
	}

	ethTxs := make([]*evmtypes.MsgEthereumTx, len(tx.GetMsgs()))
	for i, msg := range tx.GetMsgs() {
		ethTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return nil, fmt.Errorf("invalid message type %T, expected %T", msg, &evmtypes.MsgEthereumTx{})
		}
		ethTx.Hash = ethTx.AsTransaction().Hash().Hex()
		ethTxs[i] = ethTx
	}
	return ethTxs, nil
}

// EthHeaderFromTendermint: Converts a Tendermint block header to an Eth header.
func EthHeaderFromTendermint(
	header tmtypes.Header, bloom gethcoretypes.Bloom, baseFee *big.Int,
) *gethcoretypes.Header {
	txHash := gethcoretypes.EmptyRootHash
	if len(header.DataHash) == 0 {
		txHash = gethcommon.BytesToHash(header.DataHash)
	}

	time := uint64(header.Time.UTC().Unix()) // #nosec G701
	return &gethcoretypes.Header{
		ParentHash:  gethcommon.BytesToHash(header.LastBlockID.Hash.Bytes()),
		UncleHash:   gethcoretypes.EmptyUncleHash,
		Coinbase:    gethcommon.BytesToAddress(header.ProposerAddress),
		Root:        gethcommon.BytesToHash(header.AppHash),
		TxHash:      txHash,
		ReceiptHash: gethcoretypes.EmptyRootHash,
		Bloom:       bloom,
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(header.Height),
		GasLimit:    0,
		GasUsed:     0,
		Time:        time,
		Extra:       []byte{},
		MixDigest:   gethcommon.Hash{},
		Nonce:       gethcoretypes.BlockNonce{},
		BaseFee:     baseFee,
	}
}

// BlockMaxGasFromConsensusParams returns the gas limit for the current block
// from the chain consensus params.
func BlockMaxGasFromConsensusParams(
	goCtx context.Context, clientCtx client.Context, blockHeight int64,
) (int64, error) {
	tmrpcClient, ok := clientCtx.Client.(tmrpcclient.Client)
	if !ok {
		panic("incorrect tm rpc client")
	}
	resConsParams, err := tmrpcClient.ConsensusParams(goCtx, &blockHeight)
	defaultGasLimit := int64(^uint32(0)) // #nosec G701
	if err != nil {
		return defaultGasLimit, err
	}

	gasLimit := resConsParams.ConsensusParams.Block.MaxGas
	if gasLimit == -1 {
		// Sets gas limit to max uint32 to not error with javascript dev tooling
		// This -1 value indicating no block gas limit is set to max uint64 with geth hexutils
		// which errors certain javascript dev tooling which only supports up to 53 bits
		gasLimit = defaultGasLimit
	}

	return gasLimit, nil
}

// FormatBlock creates an ethereum block from a tendermint header and ethereum-formatted
// transactions.
func FormatBlock(
	header tmtypes.Header, size int, gasLimit int64,
	gasUsed *big.Int, transactions []interface{}, bloom gethcoretypes.Bloom,
	validatorAddr gethcommon.Address, baseFee *big.Int,
) map[string]interface{} {
	var transactionsRoot gethcommon.Hash
	if len(transactions) == 0 {
		transactionsRoot = gethcoretypes.EmptyRootHash
	} else {
		transactionsRoot = gethcommon.BytesToHash(header.DataHash)
	}

	result := map[string]interface{}{
		"number":           hexutil.Uint64(header.Height),
		"hash":             hexutil.Bytes(header.Hash()),
		"parentHash":       gethcommon.BytesToHash(header.LastBlockID.Hash.Bytes()),
		"nonce":            gethcoretypes.BlockNonce{},   // PoW specific
		"sha3Uncles":       gethcoretypes.EmptyUncleHash, // No uncles in Tendermint
		"logsBloom":        bloom,
		"stateRoot":        hexutil.Bytes(header.AppHash),
		"miner":            validatorAddr,
		"mixHash":          gethcommon.Hash{},
		"difficulty":       (*hexutil.Big)(big.NewInt(0)),
		"extraData":        "0x",
		"size":             hexutil.Uint64(size),
		"gasLimit":         hexutil.Uint64(gasLimit), // Static gas limit
		"gasUsed":          (*hexutil.Big)(gasUsed),
		"timestamp":        hexutil.Uint64(header.Time.Unix()),
		"transactionsRoot": transactionsRoot,
		"receiptsRoot":     gethcoretypes.EmptyRootHash,

		"uncles":          []gethcommon.Hash{},
		"transactions":    transactions,
		"totalDifficulty": (*hexutil.Big)(big.NewInt(0)),
	}

	if baseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(baseFee)
	}

	return result
}

// NewRpcTxFromMsg returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func NewRpcTxFromMsg(
	msg *evmtypes.MsgEthereumTx,
	blockHash gethcommon.Hash,
	blockNumber, index uint64,
	baseFee *big.Int,
	chainID *big.Int,
) (*RPCTransaction, error) {
	tx := msg.AsTransaction()
	return NewRpcTxFromEthTx(tx, blockHash, blockNumber, index, baseFee, chainID)
}

// NewTransactionFromData returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func NewRpcTxFromEthTx(
	tx *gethcoretypes.Transaction,
	blockHash gethcommon.Hash,
	blockNumber,
	index uint64,
	baseFee *big.Int,
	chainID *big.Int,
) (*RPCTransaction, error) {
	// Determine the signer. For replay-protected transactions, use the most
	// permissive signer, because we assume that signers are backwards-compatible
	// with old transactions. For non-protected transactions, the homestead
	// signer is used because the return value of ChainId is zero for unprotected
	// transactions.
	var signer gethcoretypes.Signer
	if tx.Protected() {
		signer = gethcoretypes.LatestSignerForChainID(tx.ChainId())
	} else {
		signer = gethcoretypes.HomesteadSigner{}
	}
	from, _ := gethcoretypes.Sender(signer, tx) // #nosec G703
	v, r, s := tx.RawSignatureValues()
	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
		ChainID:  (*hexutil.Big)(chainID),
	}
	if blockHash != (gethcommon.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}
	switch tx.Type() {
	case gethcoretypes.AccessListTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
	case gethcoretypes.DynamicFeeTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (gethcommon.Hash{}) {
			// price = min(tip, gasFeeCap - baseFee) + baseFee
			price := gethmath.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap())
			result.GasPrice = (*hexutil.Big)(price)
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
	}
	return result, nil
}

// CheckTxFee is an internal function used to check whether the fee of
// the given transaction is _reasonable_(under the cap).
func CheckTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
	// Short circuit if there is no cap for transaction fee at all.
	if cap == 0 {
		return nil
	}
	totalfee := new(big.Float).SetInt(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas)))
	oneEther := new(big.Float).SetInt(big.NewInt(gethparams.Ether))
	// quo = rounded(x/y)
	feeEth := new(big.Float).Quo(totalfee, oneEther)
	// no need to check error from parsing
	feeFloat, _ := feeEth.Float64()
	if feeFloat > cap {
		return fmt.Errorf("tx fee (%.2f ether) exceeds the configured cap (%.2f ether)", feeFloat, cap)
	}
	return nil
}

// TxExceedBlockGasLimit returns true if the tx exceeds block gas limit.
func TxExceedBlockGasLimit(res *abci.ResponseDeliverTx) bool {
	return strings.Contains(res.Log, ErrExceedBlockGasLimit)
}

// TxStateDBCommitError returns true if the evm tx commit error.
func TxStateDBCommitError(res *abci.ResponseDeliverTx) bool {
	return strings.Contains(res.Log, ErrStateDBCommit)
}

// TxSucessOrExpectedFailure returns true if the transaction was successful
// or if it failed with an ExceedBlockGasLimit error or TxStateDBCommitError error
func TxSucessOrExpectedFailure(res *abci.ResponseDeliverTx) bool {
	return res.Code == 0 || TxExceedBlockGasLimit(res) || TxStateDBCommitError(res)
}

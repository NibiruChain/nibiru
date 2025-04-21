// Copyright (c) 2023-2024 Nibi, Inc.
package rpc

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	tmtypes "github.com/cometbft/cometbft/types"

	sdkioerrors "cosmossdk.io/errors"
	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/x/evm"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
)

// ErrExceedBlockGasLimit defines the error message when tx execution exceeds the
// block gas limit. The tx fee is deducted in ante handler, so it shouldn't be
// ignored in JSON-RPC API.
const ErrExceedBlockGasLimit = "out of gas in location: block gas meter; gasWanted:"

// ErrStateDBCommit defines the error message when commit after executing EVM
// transaction, for example transfer native token to a distribution module
// account using an evm transaction. Note, the transfer amount cannot be set to
// 0, otherwise this problem will not be triggered.
const ErrStateDBCommit = "failed to commit stateDB"

// RawTxToEthTx returns a evm MsgEthereum transaction from raw tx bytes.
func RawTxToEthTx(clientCtx client.Context, txBz tmtypes.Tx) ([]*evm.MsgEthereumTx, error) {
	tx, err := clientCtx.TxConfig.TxDecoder()(txBz)
	if err != nil {
		return nil, sdkioerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	ethTxs := make([]*evm.MsgEthereumTx, len(tx.GetMsgs()))
	for i, msg := range tx.GetMsgs() {
		ethTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return nil, fmt.Errorf("invalid message type %T, expected %T", msg, &evm.MsgEthereumTx{})
		}
		ethTx.Hash = ethTx.AsTransaction().Hash().Hex()
		ethTxs[i] = ethTx
	}
	return ethTxs, nil
}

// EthHeaderFromTendermint: Converts a Tendermint block header to an Eth header.
func EthHeaderFromTendermint(
	header tmtypes.Header, bloom gethcore.Bloom, baseFeeWei *big.Int,
) *gethcore.Header {
	txHash := gethcore.EmptyRootHash
	if len(header.DataHash) == 0 {
		txHash = gethcommon.BytesToHash(header.DataHash)
	}

	time := uint64(header.Time.UTC().Unix()) // #nosec G701
	return &gethcore.Header{
		ParentHash:  gethcommon.BytesToHash(header.LastBlockID.Hash.Bytes()),
		UncleHash:   gethcore.EmptyUncleHash,
		Coinbase:    gethcommon.BytesToAddress(header.ProposerAddress),
		Root:        gethcommon.BytesToHash(header.AppHash),
		TxHash:      txHash,
		ReceiptHash: gethcore.EmptyRootHash,
		Bloom:       bloom,
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(header.Height),
		GasLimit:    0,
		GasUsed:     0,
		Time:        time,
		Extra:       []byte{},
		MixDigest:   gethcommon.Hash{},
		Nonce:       gethcore.BlockNonce{},
		BaseFee:     baseFeeWei,
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
	gasUsed *big.Int, transactions []any, bloom gethcore.Bloom,
	validatorAddr gethcommon.Address, baseFee *big.Int,
) map[string]any {
	var transactionsRoot gethcommon.Hash
	if len(transactions) == 0 {
		transactionsRoot = gethcore.EmptyRootHash
	} else {
		transactionsRoot = gethcommon.BytesToHash(header.DataHash)
	}

	result := map[string]any{
		"number":           hexutil.Uint64(header.Height),
		"hash":             hexutil.Bytes(header.Hash()),
		"parentHash":       gethcommon.BytesToHash(header.LastBlockID.Hash.Bytes()),
		"nonce":            gethcore.BlockNonce{},   // PoW specific
		"sha3Uncles":       gethcore.EmptyUncleHash, // No uncles in Tendermint
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
		"receiptsRoot":     gethcore.EmptyRootHash,

		"uncles":          []gethcommon.Hash{},
		"transactions":    transactions,
		"totalDifficulty": (*hexutil.Big)(big.NewInt(0)),
	}

	if baseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(baseFee)
	}

	return result
}

// NewRPCTxFromMsgEthTx returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func NewRPCTxFromMsgEthTx(
	msgEthTx *evm.MsgEthereumTx,
	blockHash gethcommon.Hash,
	blockNumber uint64,
	index uint64,
	baseFeeWei *big.Int,
	chainID *big.Int,
) (*EthTxJsonRPC, error) {
	var (
		tx = msgEthTx.AsTransaction()
		// Determine the signer. For replay-protected transactions, use the most
		// permissive signer, because we assume that signers are backwards-compatible
		// with old transactions. For non-protected transactions, the homestead
		// signer is used because the return value of ChainId is zero for unprotected
		// transactions.
		signer  gethcore.Signer = gethcore.HomesteadSigner{}
		v, r, s                 = tx.RawSignatureValues()
	)

	if tx.Protected() {
		signer = gethcore.LatestSignerForChainID(tx.ChainId())
	}
	from, _ := gethcore.Sender(signer, tx) // #nosec G703
	result := &EthTxJsonRPC{
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

	switch txType := tx.Type(); txType {
	case gethcore.AccessListTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
	case gethcore.DynamicFeeTxType, gethcore.BlobTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())

		// if the transaction has been mined, compute the effective gas price
		if baseFeeWei != nil && blockHash != (gethcommon.Hash{}) {
			// price = min(tip, gasFeeCap - baseFee) + baseFee
			result.GasPrice = (*hexutil.Big)(msgEthTx.EffectiveGasPriceWeiPerGas(baseFeeWei))
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
	}
	return result, nil
}

// TxIsValidEnough returns true if the transaction was successful
// or if it failed with an ExceedBlockGasLimit error or TxStateDBCommitError error
//
// Include in Block:
//   - Include successful tx
//   - Include unsuccessful tx that exceeds block gas limit
//   - Include unsuccessful tx that failed when committing changes to stateDB
//
// Exclude from Block (Not Valid Enough):
//   - Exclude unsuccessful tx with any other error but ExceedBlockGasLimit
func TxIsValidEnough(res *abci.ResponseDeliverTx) (condition bool, reason string) {
	if res.Code == 0 {
		return true, "tx succeeded"
	} else if strings.Contains(res.Log, ErrExceedBlockGasLimit) {
		return true, "tx exceeded block gas limit"
	} else if strings.Contains(res.Log, ErrStateDBCommit) {
		return true, "tx state db commit error"
	}
	return false, "unexpected failure"
}

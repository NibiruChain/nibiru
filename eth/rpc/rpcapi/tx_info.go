// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	pkgerrors "github.com/pkg/errors"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// GetTransactionByHash returns the Ethereum format transaction identified by
// Ethereum transaction hash. If the transaction is not found or has been
// discarded from a pruning node, this resolves to nil.
func (b *Backend) GetTransactionByHash(txHash gethcommon.Hash) (*rpc.EthTxJsonRPC, error) {
	res, err := b.GetTxByEthHash(txHash)
	if err != nil {
		rpcTx, pendingErr := b.getTransactionByHashPending(txHash)
		if pendingErr != nil {
			return nil, fmt.Errorf(
				"no confirmed (pending) or unconfirmed tx found: %s: %w", err, pendingErr,
			)
		}
		return rpcTx, nil
	}

	block, err := b.TendermintBlockByNumber(rpc.BlockNumber(res.Height))
	if err != nil {
		return nil, err
	}

	tx, err := b.clientCtx.TxConfig.TxDecoder()(block.Block.Txs[res.TxIndex])
	if err != nil {
		return nil, err
	}

	// the `res.MsgIndex` is inferred from tx index, should be within the bound.
	sdkMsg := tx.GetMsgs()[res.MsgIndex]
	msg, err := MsgEthereumTxFromSdkMsg(sdkMsg)
	if err != nil {
		return nil, err
	}

	blockRes, err := b.TendermintBlockResultByNumber(&block.Block.Height)
	if err != nil {
		return nil, err
	}

	if res.EthTxIndex == -1 {
		// Fallback to find tx index by iterating all valid eth transactions
		msgs := b.EthMsgsFromTendermintBlock(block, blockRes)
		for i := range msgs {
			if msgs[i].Hash == eth.EthTxHashToString(txHash) {
				if i > math.MaxInt32 {
					return nil, pkgerrors.New("tx index overflow")
				}
				res.EthTxIndex = int32(i) //#nosec G701 -- checked for int overflow already
				break
			}
		}
	}
	// if we still unable to find the eth tx index, return error, shouldn't happen.
	if res.EthTxIndex == -1 {
		return nil, pkgerrors.New("can't find index of ethereum tx")
	}

	baseFeeWei := evm.BASE_FEE_WEI
	height := uint64(res.Height)    //#nosec G701 -- checked for int overflow already
	index := uint64(res.EthTxIndex) //#nosec G701 -- checked for int overflow already
	return rpc.NewRPCTxFromMsgEthTx(
		msg,
		gethcommon.BytesToHash(block.BlockID.Hash.Bytes()),
		height,
		index,
		baseFeeWei,
		b.chainID,
	), nil
}

// getTransactionByHashPending find pending tx from mempool
func (b *Backend) getTransactionByHashPending(txHash gethcommon.Hash) (*rpc.EthTxJsonRPC, error) {
	txHashHex := txHash.Hex()
	// try to find tx in mempool
	txs, err := b.PendingTransactions()
	if err != nil {
		return nil, fmt.Errorf("error retrieving pending transactions from the mempool: %w", err)
	}

	for _, tx := range txs {
		msg, err := evm.UnwrapEthereumMsg(tx, txHash)
		if err != nil {
			// not ethereum tx
			continue
		}

		if msg.Hash == txHashHex {
			// use zero block values since it's not included in a block yet
			rpcTx := rpc.NewRPCTxFromMsgEthTx(
				msg,
				gethcommon.Hash{},
				uint64(0),
				uint64(0),
				nil,
				b.chainID,
			)
			return rpcTx, nil
		}
	}

	return nil, fmt.Errorf("no pending tx found with hash %s", txHashHex)
}

// TransactionReceipt represents the results of a transaction. TransactionReceipt
// is an extension of gethcore.Receipt, the response type for the
// "eth_getTransactionReceipt" JSON-RPC method.
// Reason being, the gethcore.Receipt struct has an incorrect JSON struct tag on one
// field and doesn't marshal JSON as expected, so we embed and extend it here.
type TransactionReceipt struct {
	gethcore.Receipt

	ContractAddress   *gethcommon.Address
	From              gethcommon.Address
	To                *gethcommon.Address
	EffectiveGasPrice *hexutil.Big
}

// MarshalJSON for [TransactionReceipt] ensures that non-receipt fields from the
// embedded [gethcore.Receipt] fields are included during JSON marshaling.
func (r *TransactionReceipt) MarshalJSON() ([]byte, error) {
	// Marshal / unmarshal gethcore.Receipt to produce map[string]interface{}
	receiptJson, err := json.Marshal(r.Receipt)
	if err != nil {
		return nil, err
	}

	var output map[string]any
	if err := json.Unmarshal(receiptJson, &output); err != nil {
		return nil, err
	}

	// Add extra (non gethcore.Receipt) fields:
	if r.ContractAddress != nil && *r.ContractAddress != (gethcommon.Address{}) {
		output["contractAddress"] = r.ContractAddress
	}
	if r.From != (gethcommon.Address{}) {
		output["from"] = r.From
	}
	if r.To != nil {
		output["to"] = r.To
	}
	if r.EffectiveGasPrice != nil {
		output["effectiveGasPrice"] = r.EffectiveGasPrice
	}
	// original marshaling of gethcore.Receipt omits type field
	output["type"] = hexutil.Uint64(r.Type)

	// delete deprecated (pre Byzantium) key which is always set to 0x and fails parsing within hardhat
	delete(output, "root")
	return json.Marshal(output)
}

func (r *TransactionReceipt) UnmarshalJSON(data []byte) error {
	// First unmarshal into a temporary map
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Unmarshal known geth Receipt
	if err := json.Unmarshal(data, &r.Receipt); err != nil {
		return fmt.Errorf("unmarshal embedded Receipt: %w", err)
	}

	// Unmarshal extra fields manually
	if v, ok := raw["from"]; ok {
		if err := json.Unmarshal(v, &r.From); err != nil {
			return fmt.Errorf("unmarshal from: %w", err)
		}
	}

	if v, ok := raw["to"]; ok {
		var to gethcommon.Address
		if string(v) != "null" {
			if err := json.Unmarshal(v, &to); err != nil {
				return fmt.Errorf("unmarshal to: %w", err)
			}
			r.To = &to
		}
	}

	if v, ok := raw["contractAddress"]; ok {
		var addr gethcommon.Address
		if err := json.Unmarshal(v, &addr); err != nil {
			return fmt.Errorf("unmarshal contractAddress: %w", err)
		}
		r.ContractAddress = &addr
	}

	if v, ok := raw["effectiveGasPrice"]; ok {
		var price hexutil.Big
		if err := json.Unmarshal(v, &price); err != nil {
			return fmt.Errorf("unmarshal effectiveGasPrice: %w", err)
		}
		r.EffectiveGasPrice = &price
	}

	if v, ok := raw["type"]; ok {
		var txType hexutil.Uint64
		if err := json.Unmarshal(v, &txType); err != nil {
			return fmt.Errorf("unmarshal type: %w", err)
		}
		r.Type = uint8(txType)
	}

	return nil
}

// GetTransactionReceipt returns the transaction receipt identified by hash.
func (b *Backend) GetTransactionReceipt(hash gethcommon.Hash) (*TransactionReceipt, error) {
	hexTx := hash.Hex()
	b.logger.Debug("eth_getTransactionReceipt", "hash", hexTx)

	res, err := b.GetTxByEthHash(hash)
	if err != nil {
		b.logger.Debug("tx not found", "hash", hexTx, "error", err.Error())
		return nil, nil
	}
	resBlock, err := b.TendermintBlockByNumber(rpc.BlockNumber(res.Height))
	if err != nil {
		b.logger.Debug("block not found", "height", res.Height, "error", err.Error())
		return nil, nil
	}
	tx, err := b.clientCtx.TxConfig.TxDecoder()(resBlock.Block.Txs[res.TxIndex])
	if err != nil {
		b.logger.Debug("decoding failed", "error", err.Error())
		return nil, fmt.Errorf("failed to decode tx: %w", err)
	}
	ethMsg := tx.GetMsgs()[res.MsgIndex].(*evm.MsgEthereumTx)

	txData, err := evm.UnpackTxData(ethMsg.Data)
	if err != nil {
		b.logger.Error("failed to unpack tx data", "error", err.Error())
		return nil, err
	}

	cumulativeGasUsed := uint64(0)
	blockRes, err := b.TendermintBlockResultByNumber(&res.Height)
	if err != nil {
		b.logger.Debug("failed to retrieve block results", "height", res.Height, "error", err.Error())
		return nil, nil
	}
	for _, txResult := range blockRes.TxsResults[0:res.TxIndex] {
		cumulativeGasUsed += uint64(txResult.GasUsed) // #nosec G701 -- checked for int overflow already
	}
	cumulativeGasUsed += res.CumulativeGasUsed

	status := gethcore.ReceiptStatusSuccessful
	if res.Failed {
		status = gethcore.ReceiptStatusFailed
	}

	chainID := b.ChainID()

	from, err := ethMsg.GetSender(chainID.ToInt())
	if err != nil {
		return nil, err
	}

	// parse tx logs from events
	msgIndex := int(res.MsgIndex) // #nosec G701 -- checked for int overflow already
	logs, err := TxLogsFromEvents(blockRes.TxsResults[res.TxIndex].Events, msgIndex)
	if err != nil {
		b.logger.Debug("failed to parse logs", "hash", hexTx, "error", err.Error())
	}

	if res.EthTxIndex == -1 {
		// Fallback to find tx index by iterating all valid eth transactions
		msgs := b.EthMsgsFromTendermintBlock(resBlock, blockRes)
		for i := range msgs {
			if msgs[i].Hash == hexTx {
				res.EthTxIndex = int32(i) // #nosec G701
				break
			}
		}
	}
	// return error if still unable to find the eth tx index
	if res.EthTxIndex == -1 {
		return nil, pkgerrors.New("can't find index of ethereum tx")
	}

	receipt := TransactionReceipt{
		Receipt: gethcore.Receipt{
			Type: ethMsg.AsTransaction().Type(),

			// Consensus fields: These fields are defined by the Etheruem Yellow Paper
			Status:            status,
			CumulativeGasUsed: cumulativeGasUsed,
			Bloom:             gethcore.BytesToBloom(gethcore.LogsBloom(logs)),
			Logs:              logs,

			// Implementation fields: These fields are added by geth when processing a transaction.
			// They are stored in the chain database.
			TxHash:  hash,
			GasUsed: res.GasUsed,

			BlockHash:        gethcommon.BytesToHash(resBlock.Block.Header.Hash()),
			BlockNumber:      big.NewInt(res.Height),
			TransactionIndex: uint(res.EthTxIndex),
		},
		ContractAddress: nil,
		From:            from,
		To:              txData.GetTo(),
	}

	if logs == nil {
		receipt.Logs = []*gethcore.Log{}
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if txData.GetTo() == nil {
		addr := crypto.CreateAddress(from, txData.GetNonce())
		receipt.ContractAddress = &addr
	}

	if dynamicTx, ok := txData.(*evm.DynamicFeeTx); ok {
		baseFeeWei := evm.BASE_FEE_WEI
		receipt.EffectiveGasPrice = (*hexutil.Big)(dynamicTx.EffectiveGasPriceWeiPerGas(baseFeeWei))
	} else {
		receipt.EffectiveGasPrice = (*hexutil.Big)(txData.GetGasPrice())
	}
	return &receipt, nil
}

// GetTransactionByBlockHashAndIndex returns the Ethereum-formatted transaction
// in the block with the given hash and specifed index in the block.
func (b *Backend) GetTransactionByBlockHashAndIndex(
	blockHash gethcommon.Hash,
	idx hexutil.Uint,
) (*rpc.EthTxJsonRPC, error) {
	resBlock, err := b.TendermintBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	return b.GetTransactionByBlockAndIndex(resBlock, idx)
}

// GetTransactionByBlockNumberAndIndex returns the Ethereum-formatted transaction
// in the block at the given block height and index within the block.
func (b *Backend) GetTransactionByBlockNumberAndIndex(
	blockNum rpc.BlockNumber,
	idx hexutil.Uint,
) (*rpc.EthTxJsonRPC, error) {
	resBlock, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}
	return b.GetTransactionByBlockAndIndex(resBlock, idx)
}

// GetTxByEthHash uses `/tx_query` to find confirmed (not pending) transaction by
// ethereum tx hash
func (b *Backend) GetTxByEthHash(hash gethcommon.Hash) (*eth.TxResult, error) {
	if b.evmTxIndexer != nil {
		return b.evmTxIndexer.GetByTxHash(hash)
	}

	// fallback to tendermint tx evmTxIndexer
	query := fmt.Sprintf("%s.%s='%s'", evm.PendingEthereumTxEvent, evm.PendingEthereumTxEventAttrEthHash, hash.Hex())

	txResult, err := b.queryTendermintTxIndexer(query, func(txs *rpc.ParsedTxs) *rpc.ParsedTx {
		return txs.GetTxByHash(hash)
	})
	if err != nil {
		return nil, sdkioerrors.Wrapf(err, "GetTxByEthHash(%s)", hash.Hex())
	}
	return txResult, nil
}

// GetTxByTxIndex uses `/tx_query` to find transaction by tx index of valid ethereum txs
func (b *Backend) GetTxByTxIndex(height int64, index uint) (*eth.TxResult, error) {
	int32Index := int32(index) // #nosec G701 -- checked for int overflow already
	if b.evmTxIndexer != nil {
		return b.evmTxIndexer.GetByBlockAndIndex(height, int32Index)
	}

	// fallback to tendermint tx evmTxIndexer
	query := fmt.Sprintf("tx.height=%d AND %s.%s=%d",
		height,
		evm.PendingEthereumTxEvent,
		evm.PendingEthereumTxEventAttrIndex,
		index,
	)
	txResult, err := b.queryTendermintTxIndexer(query, func(txs *rpc.ParsedTxs) *rpc.ParsedTx {
		return txs.GetTxByTxIndex(int(index)) // #nosec G701 -- checked for int overflow already
	})
	if err != nil {
		return nil, sdkioerrors.Wrapf(err, "GetTxByTxIndex(height=%d,index=%d)", height, index)
	}
	return txResult, nil
}

// queryTendermintTxIndexer query tx in tendermint tx evmTxIndexer
func (b *Backend) queryTendermintTxIndexer(query string, txGetter func(*rpc.ParsedTxs) *rpc.ParsedTx) (*eth.TxResult, error) {
	resTxs, err := b.clientCtx.Client.TxSearch(b.ctx, query, false, nil, nil, "")
	if err != nil {
		return nil, err
	}
	if len(resTxs.Txs) == 0 {
		return nil, pkgerrors.New("ethereum tx not found")
	}
	txResult := resTxs.Txs[0]
	isValidEnough, reason := rpc.TxIsValidEnough(&txResult.TxResult)
	if !isValidEnough {
		return nil, fmt.Errorf("invalid ethereum tx: %s", reason)
	}

	var tx sdk.Tx
	if txResult.TxResult.Code != 0 {
		// it's only needed when the tx exceeds block gas limit
		tx, err = b.clientCtx.TxConfig.TxDecoder()(txResult.Tx)
		if err != nil {
			return nil, fmt.Errorf("invalid ethereum tx: decoding failed: %w", err)
		}
	}

	return rpc.ParseTxIndexerResult(txResult, tx, txGetter)
}

// GetTransactionByBlockAndIndex is the common code shared by `GetTransactionByBlockNumberAndIndex` and `GetTransactionByBlockHashAndIndex`.
func (b *Backend) GetTransactionByBlockAndIndex(block *tmrpctypes.ResultBlock, idx hexutil.Uint) (*rpc.EthTxJsonRPC, error) {
	blockRes, err := b.TendermintBlockResultByNumber(&block.Block.Height)
	if err != nil {
		return nil, nil
	}

	var msg *evm.MsgEthereumTx
	// find in tx evmTxIndexer
	res, err := b.GetTxByTxIndex(block.Block.Height, uint(idx))
	if err == nil {
		tx, err := b.clientCtx.TxConfig.TxDecoder()(block.Block.Txs[res.TxIndex])
		if err != nil {
			b.logger.Debug("invalid ethereum tx", "height", block.Block.Header, "index", idx)
			return nil, nil
		}

		var ok bool
		// msgIndex is inferred from tx events, should be within bound.
		msg, ok = tx.GetMsgs()[res.MsgIndex].(*evm.MsgEthereumTx)
		if !ok {
			b.logger.Debug("invalid ethereum tx", "height", block.Block.Header, "index", idx)
			return nil, nil
		}
	} else {
		i := int(idx) // #nosec G701
		ethMsgs := b.EthMsgsFromTendermintBlock(block, blockRes)
		if i >= len(ethMsgs) {
			b.logger.Debug("block txs index out of bound", "index", i)
			return nil, nil
		}

		msg = ethMsgs[i]
	}

	baseFeeWei := evm.BASE_FEE_WEI
	height := uint64(block.Block.Height) // #nosec G701 -- checked for int overflow already
	index := uint64(idx)                 // #nosec G701 -- checked for int overflow already
	return rpc.NewRPCTxFromMsgEthTx(
		msg,
		gethcommon.BytesToHash(block.Block.Hash()),
		height,
		index,
		baseFeeWei,
		b.chainID,
	), nil
}

func (b *Backend) GetTransactionLogs(txHash gethcommon.Hash) ([]*gethcore.Log, error) {
	retLogs := []*gethcore.Log{}

	res, err := b.GetTxByEthHash(txHash)
	if err != nil {
		return retLogs, fmt.Errorf("tx not found: %w", err)
	} else if res.Failed {
		return retLogs, fmt.Errorf("eth tx did not succeed: txHash %v", txHash.Hex())
	}

	resBlockResult, err := b.TendermintBlockResultByNumber(&res.Height)
	if err != nil {
		return retLogs, err
	}

	// parse tx logs from events
	logs, err := TxLogsFromEvents(
		resBlockResult.TxsResults[res.TxIndex].Events,
		int(res.MsgIndex), // #nosec G701
	)
	if err != nil {
		return []*gethcore.Log{}, err
	}
	return logs, err
}

// MsgEthereumTxFromSdkMsg attempts to cast an [sdk.Msg] to [*evm.MsgEthereumTx].
// Returns an error if the type does not match.
func MsgEthereumTxFromSdkMsg(sdkMsg sdk.Msg) (*evm.MsgEthereumTx, error) {
	msg, ok := sdkMsg.(*evm.MsgEthereumTx)
	if !ok {
		wantTypeUrl := sdk.MsgTypeURL(new(evm.MsgEthereumTx))
		gotTypeUrl := sdk.MsgTypeURL(sdkMsg)
		return nil, fmt.Errorf(
			"invalid ethereum tx: failed to parse type %s and instead received %s",
			wantTypeUrl, gotTypeUrl,
		)
	}
	return msg, nil
}

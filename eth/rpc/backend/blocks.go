// Copyright (c) 2023-2024 Nibi, Inc.
package backend

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/gogoproto/proto"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/pkg/errors"
	"github.com/status-im/keycard-go/hexutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// BlockNumber returns the current block number in abci app state. Because abci
// app state could lag behind from tendermint latest block, it's more stable for
// the client to use the latest block number in abci app state than tendermint rpc.
func (b *Backend) BlockNumber() (hexutil.Uint64, error) {
	// do any grpc query, ignore the response and use the returned block height
	var header metadata.MD
	_, err := b.queryClient.Params(b.ctx, &evm.QueryParamsRequest{}, grpc.Header(&header))
	if err != nil {
		return hexutil.Uint64(0), err
	}

	blockHeightHeader := header.Get(grpctypes.GRPCBlockHeightHeader)
	if headerLen := len(blockHeightHeader); headerLen != 1 {
		return 0, fmt.Errorf("unexpected '%s' gRPC header length; got %d, expected: %d", grpctypes.GRPCBlockHeightHeader, headerLen, 1)
	}

	height, err := strconv.ParseUint(blockHeightHeader[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block height: %w", err)
	}

	if height > math.MaxInt64 {
		return 0, fmt.Errorf("block height %d is greater than max uint64", height)
	}

	return hexutil.Uint64(height), nil
}

// GetBlockByNumber returns the JSON-RPC compatible Ethereum block identified by
// block number. Depending on fullTx it either returns the full transaction
// objects or if false only the hashes of the transactions.
func (b *Backend) GetBlockByNumber(blockNum rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	resBlock, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, nil
	}

	// return if requested block height is greater than the current one
	if resBlock == nil || resBlock.Block == nil {
		return nil, nil
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		b.logger.Debug("failed to fetch block result from Tendermint", "height", blockNum, "error", err.Error())
		return nil, nil
	}

	res, err := b.RPCBlockFromTendermintBlock(resBlock, blockRes, fullTx)
	if err != nil {
		b.logger.Debug("GetEthBlockFromTendermint failed", "height", blockNum, "error", err.Error())
		return nil, err
	}

	return res, nil
}

// GetBlockByHash returns the JSON-RPC compatible Ethereum block identified by
// hash.
func (b *Backend) GetBlockByHash(hash gethcommon.Hash, fullTx bool) (map[string]interface{}, error) {
	resBlock, err := b.TendermintBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	if resBlock == nil {
		// block not found
		return nil, nil
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		b.logger.Debug("failed to fetch block result from Tendermint", "block-hash", hash.String(), "error", err.Error())
		return nil, nil
	}

	res, err := b.RPCBlockFromTendermintBlock(resBlock, blockRes, fullTx)
	if err != nil {
		b.logger.Debug("GetEthBlockFromTendermint failed", "hash", hash, "error", err.Error())
		return nil, err
	}

	return res, nil
}

// GetBlockTransactionCountByHash returns the number of Ethereum transactions in
// the block identified by hash.
func (b *Backend) GetBlockTransactionCountByHash(hash gethcommon.Hash) *hexutil.Uint {
	sc, ok := b.clientCtx.Client.(tmrpcclient.SignClient)
	if !ok {
		b.logger.Error("invalid rpc client")
	}

	block, err := sc.BlockByHash(b.ctx, hash.Bytes())
	if err != nil {
		b.logger.Debug("block not found", "hash", hash.Hex(), "error", err.Error())
		return nil
	}

	if block.Block == nil {
		b.logger.Debug("block not found", "hash", hash.Hex())
		return nil
	}

	return b.GetBlockTransactionCount(block)
}

// GetBlockTransactionCountByNumber returns the number of Ethereum transactions
// in the block identified by number.
func (b *Backend) GetBlockTransactionCountByNumber(blockNum rpc.BlockNumber) *hexutil.Uint {
	block, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		b.logger.Debug("block not found", "height", blockNum.Int64(), "error", err.Error())
		return nil
	}

	if block.Block == nil {
		b.logger.Debug("block not found", "height", blockNum.Int64())
		return nil
	}

	return b.GetBlockTransactionCount(block)
}

// GetBlockTransactionCount returns the number of Ethereum transactions in a
// given block.
func (b *Backend) GetBlockTransactionCount(block *tmrpctypes.ResultBlock) *hexutil.Uint {
	blockRes, err := b.TendermintBlockResultByNumber(&block.Block.Height)
	if err != nil {
		return nil
	}

	ethMsgs := b.EthMsgsFromTendermintBlock(block, blockRes)
	n := hexutil.Uint(len(ethMsgs))
	return &n
}

// TendermintBlockByNumber returns a Tendermint-formatted block for a given
// block number
func (b *Backend) TendermintBlockByNumber(blockNum rpc.BlockNumber) (*tmrpctypes.ResultBlock, error) {
	height := blockNum.Int64()
	if height <= 0 {
		// fetch the latest block number from the app state, more accurate than the tendermint block store state.
		n, err := b.BlockNumber()
		if err != nil {
			return nil, err
		}
		height = int64(n) //#nosec G701 -- checked for int overflow already
	}
	resBlock, err := b.clientCtx.Client.Block(b.ctx, &height)
	if err != nil {
		b.logger.Debug("tendermint client failed to get block", "height", height, "error", err.Error())
		return nil, err
	}

	if resBlock.Block == nil {
		b.logger.Debug("TendermintBlockByNumber block not found", "height", height)
		return nil, nil
	}

	return resBlock, nil
}

// TendermintBlockResultByNumber returns a Tendermint-formatted block result
// by block number
func (b *Backend) TendermintBlockResultByNumber(height *int64) (*tmrpctypes.ResultBlockResults, error) {
	sc, ok := b.clientCtx.Client.(tmrpcclient.SignClient)
	if !ok {
		return nil, errors.New("invalid rpc client")
	}
	return sc.BlockResults(b.ctx, height)
}

// TendermintBlockByHash returns a Tendermint-formatted block by block number
func (b *Backend) TendermintBlockByHash(blockHash gethcommon.Hash) (*tmrpctypes.ResultBlock, error) {
	sc, ok := b.clientCtx.Client.(tmrpcclient.SignClient)
	if !ok {
		return nil, errors.New("invalid rpc client")
	}
	resBlock, err := sc.BlockByHash(b.ctx, blockHash.Bytes())
	if err != nil {
		b.logger.Debug("tendermint client failed to get block", "blockHash", blockHash.Hex(), "error", err.Error())
		return nil, err
	}

	if resBlock == nil || resBlock.Block == nil {
		b.logger.Debug("TendermintBlockByHash block not found", "blockHash", blockHash.Hex())
		return nil, nil
	}

	return resBlock, nil
}

// BlockNumberFromTendermint returns the BlockNumber from BlockNumberOrHash
func (b *Backend) BlockNumberFromTendermint(blockNrOrHash rpc.BlockNumberOrHash) (rpc.BlockNumber, error) {
	switch {
	case blockNrOrHash.BlockHash == nil && blockNrOrHash.BlockNumber == nil:
		return rpc.EthEarliestBlockNumber, fmt.Errorf("types BlockHash and BlockNumber cannot be both nil")
	case blockNrOrHash.BlockHash != nil:
		blockNumber, err := b.BlockNumberFromTendermintByHash(*blockNrOrHash.BlockHash)
		if err != nil {
			return rpc.EthEarliestBlockNumber, err
		}
		return rpc.NewBlockNumber(blockNumber), nil
	case blockNrOrHash.BlockNumber != nil:
		return *blockNrOrHash.BlockNumber, nil
	default:
		return rpc.EthEarliestBlockNumber, nil
	}
}

// BlockNumberFromTendermintByHash returns the block height of given block hash
func (b *Backend) BlockNumberFromTendermintByHash(blockHash gethcommon.Hash) (*big.Int, error) {
	resBlock, err := b.TendermintBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	if resBlock == nil {
		return nil, errors.Errorf("block not found for hash %s", blockHash.Hex())
	}
	return big.NewInt(resBlock.Block.Height), nil
}

// EthMsgsFromTendermintBlock returns all real MsgEthereumTxs from a
// Tendermint block. It also ensures consistency over the correct txs indexes
// across RPC endpoints
func (b *Backend) EthMsgsFromTendermintBlock(
	resBlock *tmrpctypes.ResultBlock,
	blockRes *tmrpctypes.ResultBlockResults,
) []*evm.MsgEthereumTx {
	var result []*evm.MsgEthereumTx
	block := resBlock.Block

	txResults := blockRes.TxsResults

	for i, tx := range block.Txs {
		// Check if tx exists on EVM by cross checking with blockResults:
		//  - Include unsuccessful tx that exceeds block gas limit
		//  - Include unsuccessful tx that failed when committing changes to stateDB
		//  - Exclude unsuccessful tx with any other error but ExceedBlockGasLimit
		isValidEnough, reason := rpc.TxIsValidEnough(txResults[i])
		if !isValidEnough {
			b.logger.Debug(
				"invalid tx result code",
				"tm_tx_hash", eth.TmTxHashToString(tx.Hash()),
				"reason", reason,
			)
			continue
		}

		tx, err := b.clientCtx.TxConfig.TxDecoder()(tx)
		if err != nil {
			b.logger.Debug("failed to decode transaction in block", "height", block.Height, "error", err.Error())
			continue
		}

		for _, msg := range tx.GetMsgs() {
			ethMsg, ok := msg.(*evm.MsgEthereumTx)
			if !ok {
				continue
			}

			ethMsg.Hash = ethMsg.AsTransaction().Hash().Hex()
			result = append(result, ethMsg)
		}
	}
	return result
}

// HeaderByNumber returns the block header identified by height.
func (b *Backend) HeaderByNumber(blockNum rpc.BlockNumber) (*gethcore.Header, error) {
	resBlock, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}

	if resBlock == nil {
		return nil, errors.Errorf("block not found for height %d", blockNum)
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, fmt.Errorf("block result not found for height %d. %w", resBlock.Block.Height, err)
	}

	bloom, err := b.BlockBloom(blockRes)
	if err != nil {
		b.logger.Debug("HeaderByNumber BlockBloom failed", "height", resBlock.Block.Height)
	}

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		// handle the error for pruned node.
		b.logger.Error("failed to fetch Base Fee from prunned block. Check node prunning configuration", "height", resBlock.Block.Height, "error", err)
	}

	ethHeader := rpc.EthHeaderFromTendermint(resBlock.Block.Header, bloom, baseFee)
	return ethHeader, nil
}

// BlockBloom query block bloom filter from block results
func (b *Backend) BlockBloom(blockRes *tmrpctypes.ResultBlockResults) (gethcore.Bloom, error) {
	msgType := proto.MessageName((*evm.EventBlockBloom)(nil))
	for _, event := range blockRes.EndBlockEvents {
		if event.Type != msgType {
			continue
		}

		for _, attr := range event.Attributes {
			if attr.Key == evm.AttributeKeyEthereumBloom {
				return gethcore.BytesToBloom(
					hexutils.HexToBytes( // Bloom stores hex bytes
						strings.ReplaceAll(attr.Value, `"`, ""), // Unquote typed event
					),
				), nil
			}
		}
	}
	return gethcore.Bloom{}, errors.New("block bloom event is not found")
}

// RPCBlockFromTendermintBlock returns a JSON-RPC compatible Ethereum block from a
// given Tendermint block and its block result.
func (b *Backend) RPCBlockFromTendermintBlock(
	resBlock *tmrpctypes.ResultBlock,
	blockRes *tmrpctypes.ResultBlockResults,
	fullTx bool,
) (map[string]interface{}, error) {
	ethRPCTxs := []interface{}{}
	block := resBlock.Block

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		// handle the error for pruned node.
		b.logger.Error("failed to fetch Base Fee from prunned block. Check node prunning configuration", "height", block.Height, "error", err)
	}

	msgs := b.EthMsgsFromTendermintBlock(resBlock, blockRes)
	for txIndex, ethMsg := range msgs {
		if !fullTx {
			hash := gethcommon.HexToHash(ethMsg.Hash)
			ethRPCTxs = append(ethRPCTxs, hash)
			continue
		}

		tx := ethMsg.AsTransaction()
		height := uint64(block.Height) //#nosec G701 -- checked for int overflow already
		index := uint64(txIndex)       //#nosec G701 -- checked for int overflow already
		rpcTx, err := rpc.NewRPCTxFromEthTx(
			tx,
			gethcommon.BytesToHash(block.Hash()),
			height,
			index,
			baseFee,
			b.chainID,
		)
		if err != nil {
			b.logger.Debug("NewTransactionFromData for receipt failed", "hash", tx.Hash().Hex(), "error", err.Error())
			continue
		}
		ethRPCTxs = append(ethRPCTxs, rpcTx)
	}

	bloom, err := b.BlockBloom(blockRes)
	if err != nil {
		b.logger.Debug("failed to query BlockBloom", "height", block.Height, "error", err.Error())
	}

	req := &evm.QueryValidatorAccountRequest{
		ConsAddress: sdk.ConsAddress(block.Header.ProposerAddress).String(),
	}

	var validatorAccAddr sdk.AccAddress

	ctx := rpc.NewContextWithHeight(block.Height)
	res, err := b.queryClient.ValidatorAccount(ctx, req)
	if err != nil {
		b.logger.Debug(
			"failed to query validator operator address",
			"height", block.Height,
			"cons-address", req.ConsAddress,
			"error", err.Error(),
		)
		// use zero address as the validator operator address
		validatorAccAddr = sdk.AccAddress(gethcommon.Address{}.Bytes())
	} else {
		validatorAccAddr, err = sdk.AccAddressFromBech32(res.AccountAddress)
		if err != nil {
			return nil, err
		}
	}

	validatorAddr := gethcommon.BytesToAddress(validatorAccAddr)

	gasLimit, err := rpc.BlockMaxGasFromConsensusParams(ctx, b.clientCtx, block.Height)
	if err != nil {
		b.logger.Error("failed to query consensus params", "error", err.Error())
	}

	gasUsed := uint64(0)

	for _, txsResult := range blockRes.TxsResults {
		// workaround for cosmos-sdk bug. https://github.com/cosmos/cosmos-sdk/issues/10832
		if ShouldIgnoreGasUsed(txsResult) {
			// block gas limit has exceeded, other txs must have failed with same reason.
			break
		}
		gasUsed += uint64(txsResult.GetGasUsed()) // #nosec G701 -- checked for int overflow already
	}

	formattedBlock := rpc.FormatBlock(
		block.Header, block.Size(),
		gasLimit, new(big.Int).SetUint64(gasUsed),
		ethRPCTxs, bloom, validatorAddr, baseFee,
	)
	return formattedBlock, nil
}

// EthBlockByNumber returns the Ethereum Block identified by number.
func (b *Backend) EthBlockByNumber(blockNum rpc.BlockNumber) (*gethcore.Block, error) {
	resBlock, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}
	if resBlock == nil {
		// block not found
		return nil, fmt.Errorf("block not found for height %d", blockNum)
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, fmt.Errorf("block result not found for height %d", resBlock.Block.Height)
	}

	return b.EthBlockFromTendermintBlock(resBlock, blockRes)
}

// EthBlockFromTendermintBlock returns an Ethereum Block type from Tendermint block
// EthBlockFromTendermintBlock
func (b *Backend) EthBlockFromTendermintBlock(
	resBlock *tmrpctypes.ResultBlock,
	blockRes *tmrpctypes.ResultBlockResults,
) (*gethcore.Block, error) {
	block := resBlock.Block
	height := block.Height
	bloom, err := b.BlockBloom(blockRes)
	if err != nil {
		b.logger.Debug("HeaderByNumber BlockBloom failed", "height", height)
	}

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		// handle error for pruned node and log
		b.logger.Error("failed to fetch Base Fee from prunned block. Check node prunning configuration", "height", height, "error", err)
	}

	ethHeader := rpc.EthHeaderFromTendermint(block.Header, bloom, baseFee)
	msgs := b.EthMsgsFromTendermintBlock(resBlock, blockRes)

	txs := make([]*gethcore.Transaction, len(msgs))
	for i, ethMsg := range msgs {
		txs[i] = ethMsg.AsTransaction()
	}

	// TODO: add tx receipts
	ethBlock := gethcore.NewBlock(ethHeader, txs, nil, nil, trie.NewStackTrie(nil))
	return ethBlock, nil
}

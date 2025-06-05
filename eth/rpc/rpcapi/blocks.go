// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"

	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/gogoproto/proto"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
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
		return 0, fmt.Errorf("BlockNumberError: failed to query the EVM module params: %w", err)
	}

	blockHeightHeader := header.Get(grpctypes.GRPCBlockHeightHeader)
	if headerLen := len(blockHeightHeader); headerLen != 1 {
		return 0, fmt.Errorf(
			"BlockNumberError: unexpected '%s' gRPC header length; got %d, expected: %d",
			grpctypes.GRPCBlockHeightHeader, headerLen, 1)
	}

	height, err := strconv.ParseUint(blockHeightHeader[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("BlockNumberError: failed to parse block height header: %w", err)
	}

	if height > math.MaxInt64 {
		return 0, fmt.Errorf("BlockNumberError: block height %d is greater than max int64", height)
	}

	return hexutil.Uint64(height), nil
}

var ErrNilBlockSuccess = errors.New("block query succeeded, but the block was nil")

// GetBlockByNumber returns the JSON-RPC compatible Ethereum block identified by
// block number. Depending on fullTx it either returns the full transaction
// objects or if false only the hashes of the transactions.
func (b *Backend) GetBlockByNumber(
	blockNum rpc.BlockNumber,
	fullTx bool,
) (block map[string]any, err error) {
	resBlock, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}

	// return if requested block height is greater than the current one
	if resBlock == nil || resBlock.Block == nil {
		currentBlockNum, err := b.BlockNumber()
		if err != nil {
			return nil, err
			//#nosec G701 -- checked for int overflow already
		} else if blockNumI64 := blockNum.Int64(); blockNumI64 >= int64(currentBlockNum) {
			return nil, fmt.Errorf("requested block number is too high: current block %d, requested block %d",
				currentBlockNum, blockNumI64,
			)
		}
		return nil, fmt.Errorf("requested block %d: %w", blockNum, ErrNilBlockSuccess)
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, fmt.Errorf(
			"blockNumber %d: found block but failed to fetch block result: %w", blockNum, err,
		)
	}

	res, err := b.RPCBlockFromTendermintBlock(resBlock, blockRes, fullTx)
	if err != nil {
		return nil, fmt.Errorf(
			"RPCBlockFromTendermintBlock error: blockNumber %d: %w", blockNum, err)
	}

	return res, nil
}

// GetBlockByHash returns the JSON-RPC compatible Ethereum block identified by
// hash.
func (b *Backend) GetBlockByHash(
	blockHash gethcommon.Hash,
	fullTx bool,
) (block map[string]any, err error) {
	resBlock, err := b.TendermintBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}

	if resBlock == nil {
		return nil, fmt.Errorf("block not found: blockHash %s", blockHash.Hex())
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, fmt.Errorf(
			"blockHash %s: found block but failed to fetch block result: %w",
			blockHash, err,
		)
	}

	res, err := b.RPCBlockFromTendermintBlock(resBlock, blockRes, fullTx)
	if err != nil {
		return nil, fmt.Errorf("RPCBlockFromTendermintBlock error: blockHash %s: %w", blockHash, err)
	}

	return res, nil
}

// GetBlockTransactionCountByHash returns the number of Ethereum transactions in
// the block identified by hash.
func (b *Backend) GetBlockTransactionCountByHash(blockHash gethcommon.Hash) (*hexutil.Uint, error) {
	sc, ok := b.clientCtx.Client.(cmtrpcclient.SignClient)
	if !ok {
		return nil, fmt.Errorf("invalid rpc client of type %T", b.clientCtx.Client)
	}

	block, err := sc.BlockByHash(b.ctx, blockHash.Bytes())
	if err != nil {
		return nil, fmt.Errorf("block not found: hash %s: %w", blockHash, err)
	}
	if block.Block == nil {
		return nil, fmt.Errorf("block not found: hash %s: %w", blockHash, ErrNilBlockSuccess)
	}

	return b.GetBlockTransactionCount(block)
}

// GetBlockTransactionCountByNumber returns the number of Ethereum transactions
// in the block identified by number.
func (b *Backend) GetBlockTransactionCountByNumber(blockNum rpc.BlockNumber) (*hexutil.Uint, error) {
	block, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, fmt.Errorf("block not found: height %d: %w", blockNum.Int64(), err)
	}

	if block.Block == nil {
		return nil, fmt.Errorf("block not found: height %d: TendermintBlockByNumber query succeeded but returned a nil block", blockNum.Int64())
	}

	return b.GetBlockTransactionCount(block)
}

// GetBlockTransactionCount returns the number of Ethereum transactions in a
// given block.
func (b *Backend) GetBlockTransactionCount(block *tmrpctypes.ResultBlock) (*hexutil.Uint, error) {
	blockRes, err := b.TendermintBlockResultByNumber(&block.Block.Height)
	if err != nil {
		return nil, err
	}

	ethMsgs := b.EthMsgsFromTendermintBlock(block, blockRes)
	n := hexutil.Uint(len(ethMsgs))
	return &n, nil
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
		return nil, fmt.Errorf("block not found: tendermint client failed to get block %d: %w", height, err)
	}
	if resBlock.Block == nil {
		return nil, fmt.Errorf("block not found: block number %d: %w", height, ErrNilBlockSuccess)
	}

	return resBlock, nil
}

// TendermintBlockResultByNumber returns a Tendermint-formatted block result
// by block number
func (b *Backend) TendermintBlockResultByNumber(height *int64) (*tmrpctypes.ResultBlockResults, error) {
	sc, ok := b.clientCtx.Client.(cmtrpcclient.SignClient)
	if !ok {
		return nil, fmt.Errorf("invalid rpc client: type %T", b.clientCtx.Client)
	}
	blockRes, err := sc.BlockResults(b.ctx, height)
	if err != nil {
		err = fmt.Errorf("block result not found: block number %d: %w", height, err)
	}
	return blockRes, err
}

// TendermintBlockByHash returns a Tendermint-formatted block by block number
func (b *Backend) TendermintBlockByHash(blockHash gethcommon.Hash) (*tmrpctypes.ResultBlock, error) {
	sc, ok := b.clientCtx.Client.(cmtrpcclient.SignClient)
	if !ok {
		return nil, fmt.Errorf("TendermintBlockByHash: invalid RPC client: type %T", b.clientCtx.Client)
	}
	resBlock, err := sc.BlockByHash(b.ctx, blockHash.Bytes())
	if err != nil {
		return nil, fmt.Errorf(
			"block not found: blockHash %s: %w", blockHash.Hex(), err,
		)
	}

	if resBlock == nil || resBlock.Block == nil {
		return nil, fmt.Errorf(
			"block not found: blockHash %s: %w",
			blockHash.Hex(), ErrNilBlockSuccess,
		)
	}

	return resBlock, nil
}

// BlockNumberFromTendermint parses the [rpc.BlockNumber] from the given
// [rpc.BlockNumberOrHash].
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
		return nil, fmt.Errorf("block not found for hash %s", blockHash.Hex())
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

	for i, tx := range block.Txs {
		// Check if tx exists on EVM by cross checking with blockResults:
		//  - Include unsuccessful tx that exceeds block gas limit
		//  - Include unsuccessful tx that failed when committing changes to stateDB
		//  - Exclude unsuccessful tx with any other error but ExceedBlockGasLimit
		isValidEnough, reason := rpc.TxIsValidEnough(blockRes.TxsResults[i])
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
			b.logger.Debug(
				"failed to decode transaction in block", "height",
				block.Height, "error", err.Error())
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

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, err
	}

	bloom := b.BlockBloom(blockRes)
	baseFeeWei := evm.BASE_FEE_WEI

	ethHeader := rpc.EthHeaderFromTendermint(resBlock.Block.Header, bloom, baseFeeWei)
	return ethHeader, nil
}

// BlockBloom query block bloom filter from block results
func (b *Backend) BlockBloom(blockRes *tmrpctypes.ResultBlockResults) (bloom gethcore.Bloom) {
	if blockRes == nil || len(blockRes.EndBlockEvents) == 0 {
		return bloom
	}
	msgType := proto.MessageName((*evm.EventBlockBloom)(nil))
	for _, event := range blockRes.EndBlockEvents {
		if event.Type != msgType {
			continue
		}
		blockBloomEvent, err := evm.EventBlockBloomFromABCIEvent(event)
		if err != nil {
			continue
		}
		return gethcore.BytesToBloom(hexutils.HexToBytes(blockBloomEvent.Bloom))
	}

	// Suppressing error as it is expected to be missing for pruned node or for blocks before evm
	return gethcore.Bloom{}
}

// RPCBlockFromTendermintBlock returns a JSON-RPC compatible Ethereum block from a
// given Tendermint block and its block result.
func (b *Backend) RPCBlockFromTendermintBlock(
	resBlock *tmrpctypes.ResultBlock,
	blockRes *tmrpctypes.ResultBlockResults,
	fullTx bool,
) (map[string]any, error) {
	ethRPCTxs := []any{}
	block := resBlock.Block
	baseFeeWei := evm.BASE_FEE_WEI

	msgs := b.EthMsgsFromTendermintBlock(resBlock, blockRes)
	for txIndex, ethMsg := range msgs {
		if !fullTx {
			hash := gethcommon.HexToHash(ethMsg.Hash)
			ethRPCTxs = append(ethRPCTxs, hash)
			continue
		}

		height := uint64(block.Height) //#nosec G701 -- checked for int overflow already
		index := uint64(txIndex)       //#nosec G701 -- checked for int overflow already
		rpcTx := rpc.NewRPCTxFromMsgEthTx(
			ethMsg,
			gethcommon.BytesToHash(block.Hash()),
			height,
			index,
			baseFeeWei,
			b.chainID,
		)
		ethRPCTxs = append(ethRPCTxs, rpcTx)
	}

	bloom := b.BlockBloom(blockRes)

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
		ethRPCTxs, bloom, validatorAddr, baseFeeWei,
	)
	return formattedBlock, nil
}

// EthBlockByNumber returns the Ethereum Block identified by number.
func (b *Backend) EthBlockByNumber(blockNum rpc.BlockNumber) (*gethcore.Block, error) {
	resBlock, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, err
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
	bloom := b.BlockBloom(blockRes)
	baseFeeWei := evm.BASE_FEE_WEI

	ethHeader := rpc.EthHeaderFromTendermint(block.Header, bloom, baseFeeWei)
	msgs := b.EthMsgsFromTendermintBlock(resBlock, blockRes)

	txs := make([]*gethcore.Transaction, len(msgs))
	for i, ethMsg := range msgs {
		txs[i] = ethMsg.AsTransaction()
	}

	body := &gethcore.Body{
		Transactions: txs,
		Uncles:       []*gethcore.Header{},     // unused
		Withdrawals:  []*gethcore.Withdrawal{}, // unused: Specific to Etheruem mainnet
	}

	// TODO: feat(evm-backend): Add tx receipts in gethcore.NewBlock for the
	// EthBlockFromTendermintBlock function.
	// ticket: https://github.com/NibiruChain/nibiru/issues/2282

	// TODO: feat: See if we can simulate Trie behavior on CometBFT.

	var (
		receipts                     = []*gethcore.Receipt{}
		hasher   gethcore.TrieHasher = trie.NewStackTrie(nil)
	)
	ethBlock := gethcore.NewBlock(ethHeader, body, receipts, hasher)
	return ethBlock, nil
}

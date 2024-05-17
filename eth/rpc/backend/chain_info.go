// Copyright (c) 2023-2024 Nibi, Inc.
package backend

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/rpc"
	"github.com/NibiruChain/nibiru/x/evm"
)

// ChainID is the EIP-155 replay-protection chain id for the current ethereum chain config.
func (b *Backend) ChainID() (*hexutil.Big, error) {
	eip155ChainID, err := eth.ParseEthChainID(b.clientCtx.ChainID)
	if err != nil {
		panic(err)
	}
	// if current block is at or past the EIP-155 replay-protection fork block, return chainID from config
	bn, err := b.BlockNumber()
	if err != nil {
		b.logger.Debug("failed to fetch latest block number", "error", err.Error())
		return (*hexutil.Big)(eip155ChainID), nil
	}

	if config := b.ChainConfig(); config.IsEIP155(new(big.Int).SetUint64(uint64(bn))) {
		return (*hexutil.Big)(config.ChainID), nil
	}

	return nil, fmt.Errorf("chain not synced beyond EIP-155 replay-protection fork block")
}

// ChainConfig returns the latest ethereum chain configuration
func (b *Backend) ChainConfig() *params.ChainConfig {
	params, err := b.queryClient.Params(b.ctx, &evm.QueryParamsRequest{})
	if err != nil {
		return nil
	}

	return params.Params.ChainConfig.EthereumConfig(b.chainID)
}

// BaseFee returns the base fee tracked by the Fee Market module.
// If the base fee is not enabled globally, the query returns nil.
// If the London hard fork is not activated at the current height, the query will
// return nil.
func (b *Backend) BaseFee(
	blockRes *tmrpctypes.ResultBlockResults,
) (baseFee *big.Int, err error) {
	// return BaseFee if London hard fork is activated and feemarket is enabled
	res, err := b.queryClient.BaseFee(rpc.NewContextWithHeight(blockRes.Height), &evm.QueryBaseFeeRequest{})
	if err != nil || res.BaseFee == nil {
		baseFee = nil
		// TODO: feat: dynamic fee handling on events
		return baseFee, nil
	}
	return res.BaseFee.BigInt(), nil
}

// CurrentHeader returns the latest block header
// This will return error as per node configuration
// if the ABCI responses are discarded ('discard_abci_responses' config param)
func (b *Backend) CurrentHeader() (*gethcore.Header, error) {
	return b.HeaderByNumber(rpc.EthLatestBlockNumber)
}

// PendingTransactions returns the transactions that are in the transaction pool
// and have a from address that is one of the accounts this node manages.
func (b *Backend) PendingTransactions() ([]*sdk.Tx, error) {
	mc, ok := b.clientCtx.Client.(tmrpcclient.MempoolClient)
	if !ok {
		return nil, errors.New("invalid rpc client")
	}

	res, err := mc.UnconfirmedTxs(b.ctx, nil)
	if err != nil {
		return nil, err
	}

	result := make([]*sdk.Tx, 0, len(res.Txs))
	for _, txBz := range res.Txs {
		tx, err := b.clientCtx.TxConfig.TxDecoder()(txBz)
		if err != nil {
			return nil, err
		}
		result = append(result, &tx)
	}

	return result, nil
}

// GetCoinbase is the address that staking rewards will be send to (alias for Etherbase).
func (b *Backend) GetCoinbase() (sdk.AccAddress, error) {
	node, err := b.clientCtx.GetNode()
	if err != nil {
		return nil, err
	}

	status, err := node.Status(b.ctx)
	if err != nil {
		return nil, err
	}

	req := &evm.QueryValidatorAccountRequest{
		ConsAddress: sdk.ConsAddress(status.ValidatorInfo.Address).String(),
	}

	res, err := b.queryClient.ValidatorAccount(b.ctx, req)
	if err != nil {
		return nil, err
	}

	address, _ := sdk.AccAddressFromBech32(res.AccountAddress) // #nosec G703
	return address, nil
}

// FeeHistory returns data relevant for fee estimation based on the specified range of blocks.
func (b *Backend) FeeHistory(
	userBlockCount gethrpc.DecimalOrHex, // number blocks to fetch, maximum is 100
	lastBlock gethrpc.BlockNumber, // the block to start search , to oldest
	rewardPercentiles []float64, // percentiles to fetch reward
) (*rpc.FeeHistoryResult, error) {
	blockEnd := int64(lastBlock) //#nosec G701 -- checked for int overflow already

	if blockEnd < 0 {
		blockNumber, err := b.BlockNumber()
		if err != nil {
			return nil, err
		}
		blockEnd = int64(blockNumber) //#nosec G701 -- checked for int overflow already
	}

	blocks := int64(userBlockCount)                     // #nosec G701 -- checked for int overflow already
	maxBlockCount := int64(b.cfg.JSONRPC.FeeHistoryCap) // #nosec G701 -- checked for int overflow already
	if blocks > maxBlockCount {
		return nil, fmt.Errorf("FeeHistory user block count %d higher than %d", blocks, maxBlockCount)
	}

	if blockEnd+1 < blocks {
		blocks = blockEnd + 1
	}
	// Ensure not trying to retrieve before genesis.
	blockStart := blockEnd + 1 - blocks
	oldestBlock := (*hexutil.Big)(big.NewInt(blockStart))

	// prepare space
	reward := make([][]*hexutil.Big, blocks)
	rewardCount := len(rewardPercentiles)
	for i := 0; i < int(blocks); i++ {
		reward[i] = make([]*hexutil.Big, rewardCount)
	}

	thisBaseFee := make([]*hexutil.Big, blocks+1)
	thisGasUsedRatio := make([]float64, blocks)

	// rewards should only be calculated if reward percentiles were included
	calculateRewards := rewardCount != 0

	// fetch block
	for blockID := blockStart; blockID <= blockEnd; blockID++ {
		index := int32(blockID - blockStart) // #nosec G701
		// tendermint block
		tendermintblock, err := b.TendermintBlockByNumber(rpc.BlockNumber(blockID))
		if tendermintblock == nil {
			return nil, err
		}

		// eth block
		ethBlock, err := b.GetBlockByNumber(rpc.BlockNumber(blockID), true)
		if ethBlock == nil {
			return nil, err
		}

		// tendermint block result
		tendermintBlockResult, err := b.TendermintBlockResultByNumber(&tendermintblock.Block.Height)
		if tendermintBlockResult == nil {
			b.logger.Debug("block result not found", "height", tendermintblock.Block.Height, "error", err.Error())
			return nil, err
		}

		oneFeeHistory := rpc.OneFeeHistory{}
		err = b.processBlock(tendermintblock, &ethBlock, rewardPercentiles, tendermintBlockResult, &oneFeeHistory)
		if err != nil {
			return nil, err
		}

		// copy
		thisBaseFee[index] = (*hexutil.Big)(oneFeeHistory.BaseFee)
		thisBaseFee[index+1] = (*hexutil.Big)(oneFeeHistory.NextBaseFee)
		thisGasUsedRatio[index] = oneFeeHistory.GasUsedRatio
		if calculateRewards {
			for j := 0; j < rewardCount; j++ {
				reward[index][j] = (*hexutil.Big)(oneFeeHistory.Reward[j])
				if reward[index][j] == nil {
					reward[index][j] = (*hexutil.Big)(big.NewInt(0))
				}
			}
		}
	}

	feeHistory := rpc.FeeHistoryResult{
		OldestBlock:  oldestBlock,
		BaseFee:      thisBaseFee,
		GasUsedRatio: thisGasUsedRatio,
	}

	if calculateRewards {
		feeHistory.Reward = reward
	}

	return &feeHistory, nil
}

// SuggestGasTipCap: Not yet supported. Returns 0 as the suggested tip cap. After
// implementing tx prioritization, this function can come to life.
func (b *Backend) SuggestGasTipCap(baseFee *big.Int) (*big.Int, error) {
	maxDelta := big.NewInt(0)
	return maxDelta, nil
}

func DefaultMinGasPrice() sdkmath.LegacyDec { return sdkmath.LegacyZeroDec() }

// GlobalMinGasPrice returns the minimum gas price for all nodes. This is
// distinct from the individual configuration set by the validator set.
func (b *Backend) GlobalMinGasPrice() (sdkmath.LegacyDec, error) {
	// TODO: feat(eth): dynamic fees
	return DefaultMinGasPrice(), nil
}

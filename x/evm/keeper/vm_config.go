// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

func (k *Keeper) GetEVMConfig(
	ctx sdk.Context, proposerAddress sdk.ConsAddress, chainID *big.Int,
) (*statedb.EVMConfig, error) {
	params := k.GetParams(ctx)
	ethCfg := evm.EthereumConfig(chainID)

	// get the coinbase address from the block proposer
	coinbase, err := k.GetCoinbaseAddress(ctx, proposerAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain coinbase address")
	}

	baseFee := k.BaseFeeMicronibiPerGas(ctx)
	return &statedb.EVMConfig{
		Params:      params,
		ChainConfig: ethCfg,
		CoinBase:    coinbase,
		BaseFee:     baseFee,
	}, nil
}

// TxConfig loads `TxConfig` from current transient storage
func (k *Keeper) TxConfig(
	ctx sdk.Context, txHash common.Hash,
) statedb.TxConfig {
	return statedb.NewTxConfig(
		common.BytesToHash(ctx.HeaderHash()), // BlockHash
		txHash,                               // TxHash
		uint(k.EvmState.BlockTxIndex.GetOr(ctx, 0)), // TxIndex
		uint(k.EvmState.BlockLogSize.GetOr(ctx, 0)), // LogIndex
	)
}

// VMConfig creates an EVM configuration from the debug setting and the extra
// EIPs enabled on the module parameters. The config generated uses the default
// JumpTable from the EVM.
func (k Keeper) VMConfig(
	ctx sdk.Context, _ core.Message, cfg *statedb.EVMConfig, tracer vm.EVMLogger,
) vm.Config {
	var debug bool
	if _, ok := tracer.(evm.NoOpTracer); !ok {
		debug = true
	}

	return vm.Config{
		Debug:     debug,
		Tracer:    tracer,
		NoBaseFee: false,
		ExtraEips: cfg.Params.EIPs(),
	}
}

// GetCoinbaseAddress returns the block proposer's validator operator address.
func (k Keeper) GetCoinbaseAddress(ctx sdk.Context, proposerAddress sdk.ConsAddress) (common.Address, error) {
	validator, found := k.stakingKeeper.GetValidatorByConsAddr(ctx, ParseProposerAddr(ctx, proposerAddress))
	if !found {
		return common.Address{}, errors.Wrapf(
			stakingtypes.ErrNoValidatorFound,
			"failed to retrieve validator from block proposer address %s",
			proposerAddress.String(),
		)
	}

	coinbase := common.BytesToAddress(validator.GetOperator())
	return coinbase, nil
}

// ParseProposerAddr returns current block proposer's address when provided
// proposer address is empty.
func ParseProposerAddr(
	ctx sdk.Context, proposerAddress sdk.ConsAddress,
) sdk.ConsAddress {
	if len(proposerAddress) == 0 {
		proposerAddress = ctx.BlockHeader().ProposerAddress
	}
	return proposerAddress
}

// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

func (k *Keeper) GetEVMConfig(ctx sdk.Context) statedb.EVMConfig {
	return statedb.EVMConfig{
		Params:        k.GetParams(ctx),
		ChainConfig:   evm.EthereumConfig(appconst.GetEthChainID(ctx.ChainID())),
		BlockCoinbase: k.GetCoinbaseAddress(ctx),
		BaseFeeWei:    k.BaseFeeWeiPerGas(ctx),
	}
}

// TxConfig loads `TxConfig` from current transient storage
func (k *Keeper) TxConfig(
	ctx sdk.Context, txHash common.Hash,
) statedb.TxConfig {
	return statedb.TxConfig{
		BlockHash: common.BytesToHash(ctx.HeaderHash()),
		TxHash:    txHash,
		TxIndex:   uint(k.EvmState.BlockTxIndex.GetOr(ctx, 0)),
		LogIndex:  uint(k.EvmState.BlockLogSize.GetOr(ctx, 0)),
	}
}

// VMConfig creates an EVM configuration from the debug setting and the extra
// EIPs enabled on the module parameters. The config generated uses the default
// JumpTable from the EVM.
func (k Keeper) VMConfig(
	ctx sdk.Context, cfg *statedb.EVMConfig, tracer *tracing.Hooks,
) vm.Config {
	return vm.Config{
		Tracer:    tracer,
		NoBaseFee: false,
		ExtraEips: cfg.Params.EIPs(),
	}
}

// GetCoinbaseAddress returns the block proposer's validator operator address.
// In Ethereum, the coinbase (or "benficiary") is the address that proposed the
// current block. It corresponds to the [COINBASE op code].
//
// [COINBASE op code]: https://ethereum.org/en/developers/docs/evm/opcodes/
func (k Keeper) GetCoinbaseAddress(ctx sdk.Context) common.Address {
	proposerAddress := sdk.ConsAddress(ctx.BlockHeader().ProposerAddress)
	validator, found := k.stakingKeeper.GetValidatorByConsAddr(ctx, proposerAddress)
	if !found {
		// should never happen, but just in case, return an empty address
		// we don't really care about the coinbase adresss since we're PoS and not PoW
		return common.Address{}
	}

	return common.BytesToAddress(validator.GetOperator())
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

// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/x/evm"
	evmkeeper "github.com/NibiruChain/nibiru/x/evm/keeper"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

// EVMKeeper defines the expected keeper interface used on the AnteHandler
type EVMKeeper interface {
	statedb.Keeper

	NewEVM(ctx sdk.Context, msg core.Message, cfg *statedb.EVMConfig, tracer vm.EVMLogger, stateDB vm.StateDB) *vm.EVM
	DeductTxCostsFromUserBalance(ctx sdk.Context, fees sdk.Coins, from common.Address) error
	GetEvmGasBalance(ctx sdk.Context, addr common.Address) *big.Int
	ResetTransientGasUsed(ctx sdk.Context)
	GetParams(ctx sdk.Context) evm.Params

	EVMState() evmkeeper.EvmState
	EthChainID(ctx sdk.Context) *big.Int
	GetBaseFee(ctx sdk.Context) *big.Int
}

type protoTxProvider interface {
	GetProtoTx() *tx.Tx
}

// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	evmkeeper "github.com/NibiruChain/nibiru/x/evm/keeper"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// EVMKeeper defines the expected keeper interface used on the AnteHandler
type EVMKeeper interface {
	statedb.Keeper

	NewEVM(ctx sdk.Context, msg core.Message, cfg *statedb.EVMConfig, tracer vm.EVMLogger, stateDB vm.StateDB) *vm.EVM
	DeductTxCostsFromUserBalance(ctx sdk.Context, fees sdk.Coins, from common.Address) error
	GetEvmGasBalance(ctx sdk.Context, addr common.Address) *big.Int
	ResetTransientGasUsed(ctx sdk.Context)
	GetParams(ctx sdk.Context) types.Params

	EVMState() evmkeeper.EvmState
}

type protoTxProvider interface {
	GetProtoTx() *tx.Tx
}

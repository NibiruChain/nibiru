// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"github.com/cosmos/cosmos-sdk/types/tx"

	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
)

type EVMKeeper = evmkeeper.Keeper

type protoTxProvider interface {
	GetProtoTx() *tx.Tx
}

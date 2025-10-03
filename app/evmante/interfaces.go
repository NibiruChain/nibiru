// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"github.com/cosmos/cosmos-sdk/types/tx"

	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

type EVMKeeper = evmstate.Keeper

type protoTxProvider interface {
	GetProtoTx() *tx.Tx
}

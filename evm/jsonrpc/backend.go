// Copyright (c) 2023-2024 Nibi, Inc.
package jsonrpc

import (
	"context"
	"math/big"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"

	"github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm/rpc"
)

// Backend implements the functionality shared within Ethereum JSON-RPC namespaces
// as defined by [EIP-1474].
//
// [EIP-1474]: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1474.md
type Backend struct {
	ctx                 context.Context
	clientCtx           client.Context
	queryClient         *rpc.QueryClient // gRPC query client
	logger              log.Logger
	chainID             *big.Int
	cfg                 config.Config
	allowUnprotectedTxs bool
	evmTxIndexer        eth.EVMTxIndexer
}

// NewBackend creates a new Backend instance for Ethereum JSON-RPC namespaces.
func NewBackend(
	ctx *server.Context,
	logger log.Logger,
	clientCtx client.Context,
	allowUnprotectedTxs bool,
	evmTxIndexer eth.EVMTxIndexer,
) *Backend {
	chainID := eth.ParseEthChainID(clientCtx.ChainID)
	appConf, err := config.GetConfig(ctx.Viper)
	if err != nil {
		panic(err)
	}

	return &Backend{
		ctx:                 context.Background(),
		clientCtx:           clientCtx,
		queryClient:         rpc.NewQueryClient(clientCtx),
		logger:              logger.With("module", "backend"),
		chainID:             chainID,
		cfg:                 appConf,
		allowUnprotectedTxs: allowUnprotectedTxs,
		evmTxIndexer:        evmTxIndexer,
	}
}

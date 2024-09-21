// Copyright (c) 2023-2024 Nibi, Inc.
package backend

import (
	"context"
	"math/big"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
)

// Backend implements implements the functionality shared within ethereum namespaces
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

// NewBackend creates a new Backend instance for cosmos and ethereum namespaces
func NewBackend(
	ctx *server.Context,
	logger log.Logger,
	clientCtx client.Context,
	allowUnprotectedTxs bool,
	evmTxIndexer eth.EVMTxIndexer,
) *Backend {
	chainID, err := eth.ParseEthChainID(clientCtx.ChainID)
	if err != nil {
		panic(err)
	}

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

// CosmosBackend: Currently unused. Backend functionality for the shared
// "cosmos" RPC namespace. Implements [BackendI] in combination with [Backend].
// TODO: feat(eth): Implement the cosmos JSON-RPC defined by Wallet Connect V2:
// https://docs.walletconnect.com/2.0/json-rpc/cosmos.
type CosmosBackend interface {
	// TODO: GetAccounts()
	// TODO: SignDirect()
	// TODO: SignAmino()
}

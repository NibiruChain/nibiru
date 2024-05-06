// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/server"

	"github.com/NibiruChain/nibiru/eth/rpc/backend"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/cometbft/cometbft/libs/log"
)

// MinerAPI is the private miner prefixed set of APIs in the Miner JSON-RPC spec.
type MinerAPI struct {
	ctx     *server.Context
	logger  log.Logger
	backend backend.EVMBackend
}

// NewImplMinerAPI creates an instance of the Miner API.
func NewImplMinerAPI(
	ctx *server.Context,
	backend backend.EVMBackend,
) *MinerAPI {
	return &MinerAPI{
		ctx:     ctx,
		logger:  ctx.Logger.With("api", "miner"),
		backend: backend,
	}
}

// SetEtherbase sets the etherbase of the miner
func (api *MinerAPI) SetEtherbase(etherbase common.Address) bool {
	api.logger.Debug("miner_setEtherbase")
	return api.backend.SetEtherbase(etherbase)
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (api *MinerAPI) SetGasPrice(gasPrice hexutil.Big) bool {
	api.logger.Info(api.ctx.Viper.ConfigFileUsed())
	return api.backend.SetGasPrice(gasPrice)
}

// ------------------------------------------------
// Unsupported functions on the Miner API
// ------------------------------------------------

// GetHashrate returns the current hashrate for local CPU miner and remote miner.
// Unsupported in Ethermint
func (api *MinerAPI) GetHashrate() uint64 {
	api.logger.Debug("miner_getHashrate")
	api.logger.Debug("Unsupported rpc function: miner_getHashrate")
	return 0
}

// SetExtra sets the extra data string that is included when this miner mines a block.
// Unsupported in Ethermint
func (api *MinerAPI) SetExtra(_ string) (bool, error) {
	api.logger.Debug("miner_setExtra")
	api.logger.Debug("Unsupported rpc function: miner_setExtra")
	return false, errors.New("unsupported rpc function: miner_setExtra")
}

// SetGasLimit sets the gaslimit to target towards during mining.
// Unsupported in Ethermint
func (api *MinerAPI) SetGasLimit(_ hexutil.Uint64) bool {
	api.logger.Debug("miner_setGasLimit")
	api.logger.Debug("Unsupported rpc function: miner_setGasLimit")
	return false
}

// Start starts the miner with the given number of threads. If threads is nil,
// the number of workers started is equal to the number of logical CPUs that are
// usable by this process. If mining is already running, this method adjust the
// number of threads allowed to use and updates the minimum price required by the
// transaction pool.
// Unsupported in Ethermint
func (api *MinerAPI) Start(_ *int) error {
	api.logger.Debug("miner_start")
	api.logger.Debug("Unsupported rpc function: miner_start")
	return errors.New("unsupported rpc function: miner_start")
}

// Stop terminates the miner, both at the consensus engine level as well as at
// the block creation level.
// Unsupported in Ethermint
func (api *MinerAPI) Stop() {
	api.logger.Debug("miner_stop")
	api.logger.Debug("Unsupported rpc function: miner_stop")
}

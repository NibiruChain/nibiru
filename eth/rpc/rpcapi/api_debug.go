package rpcapi

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime" // #nosec G702
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"

	"github.com/cosmos/cosmos-sdk/server"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	getheth "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// DebugAPI is the collection of tracing APIs exposed over the private debugging
// endpoint.
type DebugAPI struct {
	ctx     *server.Context
	logger  log.Logger
	backend *Backend
}

// NewImplDebugAPI creates a new API definition for the tracing methods of the
// Ethereum service.
func NewImplDebugAPI(
	ctx *server.Context,
	backend *Backend,
) *DebugAPI {
	return &DebugAPI{
		ctx:     ctx,
		logger:  ctx.Logger.With("module", "debug"),
		backend: backend,
	}
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (a *DebugAPI) TraceTransaction(
	hash common.Hash,
	config *evm.TraceConfig,
) (any, error) {
	a.logger.Debug("debug_traceTransaction", "hash", hash)
	return a.backend.TraceTransaction(hash, config)
}

// TraceBlockByNumber returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (a *DebugAPI) TraceBlockByNumber(
	height rpc.BlockNumber,
	config *evm.TraceConfig,
) ([]*evm.TxTraceResult, error) {
	a.logger.Debug("debug_traceBlockByNumber", "height", height)
	if height == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	// Get Tendermint Block
	resBlock, err := a.backend.TendermintBlockByNumber(height)
	if err != nil {
		err = fmt.Errorf("%s { blockHeight: %d }", err, height)
		a.logger.Debug("get block failed", "error", err.Error())
		return nil, err
	}

	return a.backend.TraceBlock(rpc.BlockNumber(resBlock.Block.Height), config, resBlock)
}

// TraceBlockByHash returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (a *DebugAPI) TraceBlockByHash(
	hash common.Hash,
	config *evm.TraceConfig,
) ([]*evm.TxTraceResult, error) {
	a.logger.Debug("debug_traceBlockByHash", "hash", hash)
	// Get Tendermint Block
	resBlock, err := a.backend.TendermintBlockByHash(hash)
	if err != nil {
		a.logger.Debug("get block failed", "hash", hash.Hex(), "error", err.Error())
		return nil, err
	}

	if resBlock == nil || resBlock.Block == nil {
		a.logger.Debug("block not found", "hash", hash.Hex())
		return nil, errors.New("block not found")
	}

	return a.backend.TraceBlock(rpc.BlockNumber(resBlock.Block.Height), config, resBlock)
}

// TraceCall implements eth debug_traceCall method which lets you run an eth_call
// within the context of the given block execution using the final state of parent block as the base.
// Method returns the structured logs created during the execution of EVM.
// The method returns the same output as debug_traceTransaction.
// https://geth.ethereum.org/docs/interacting-with-geth/rpc/ns-debug#debugtracecall
func (a *DebugAPI) TraceCall(
	args evm.JsonTxArgs,
	blockNrOrHash rpc.BlockNumberOrHash,
	config *evm.TraceConfig,
) (any, error) {
	a.logger.Debug("debug_traceCall", args.String(), "block number or hash", blockNrOrHash)

	// Get Tendermint Block
	resBlock, err := a.backend.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		a.logger.Debug("get block failed", "blockNrOrHash", blockNrOrHash, "error", err.Error())
		return nil, err
	}
	return a.backend.TraceCall(args, resBlock, config)
}

// GetHeaderRlp retrieves the RLP encoded for of a single header.
func (a *DebugAPI) GetHeaderRlp(number uint64) (hexutil.Bytes, error) {
	header, err := a.backend.HeaderByNumber(rpc.BlockNumber(number))
	if err != nil {
		return nil, err
	}

	return rlp.EncodeToBytes(header)
}

// GetBlockRlp retrieves the RLP encoded for of a single block.
func (a *DebugAPI) GetBlockRlp(number uint64) (hexutil.Bytes, error) {
	block, err := a.backend.EthBlockByNumber(rpc.BlockNumber(number))
	if err != nil {
		return nil, err
	}

	return rlp.EncodeToBytes(block)
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (a *DebugAPI) PrintBlock(number uint64) (string, error) {
	block, err := a.backend.EthBlockByNumber(rpc.BlockNumber(number))
	if err != nil {
		return "", err
	}

	return spew.Sdump(block), nil
}

// --------------------------------------------------------------------------
// Code beyond this point falls under the disbaled portion of the debug namespace
// --------------------------------------------------------------------------

func ErrNotImplemented(method string) error {
	return fmt.Errorf("method %q is intentionally disabled or not implemented", method)
}

func noOpMethod(method string) string { return fmt.Sprintf("%v (no-op)", method) }

// BlockProfile turns on goroutine profiling for nsec seconds and writes profile data to file.
func (a *DebugAPI) BlockProfile(file string, nsec uint) error {
	methodName := "debug_blockProfile"
	a.logger.Debug(noOpMethod(methodName), "file", file, "nsec", nsec)
	return ErrNotImplemented(methodName)
}

// CpuProfile turns on CPU profiling for nsec seconds and writes profile data to file.
func (a *DebugAPI) CpuProfile(file string, nsec uint) error { //nolint: golint, stylecheck, revive
	methodName := "debug_cpuProfile"
	a.logger.Debug(noOpMethod(methodName), "file", file, "nsec", nsec)
	return ErrNotImplemented(methodName)
}

// GcStats returns GC statistics.
func (a *DebugAPI) GcStats() *debug.GCStats {
	a.logger.Debug("debug_gcStats")
	s := new(debug.GCStats)
	debug.ReadGCStats(s)
	return s
}

// GoTrace turns on tracing for nsec seconds and writes trace data to file.
func (a *DebugAPI) GoTrace(file string, nsec uint) error {
	methodName := "debug_goTrace"
	a.logger.Debug(noOpMethod(methodName), "file", file, "nsec", nsec)
	return ErrNotImplemented(methodName)
}

// MemStats returns detailed runtime memory statistics.
func (a *DebugAPI) MemStats() *runtime.MemStats {
	a.logger.Debug("debug_memStats")
	s := new(runtime.MemStats)
	runtime.ReadMemStats(s)
	return s
}

// SetBlockProfileRate sets the rate of goroutine block profile data collection.
// rate 0 disables block profiling.
func (a *DebugAPI) SetBlockProfileRate(rate int) {
	a.logger.Debug(noOpMethod("debug_setBlockProfileRate"), "rate", rate)
}

// Stacks returns a printed representation of the stacks of all goroutines.
func (a *DebugAPI) Stacks() string {
	a.logger.Debug(noOpMethod("debug_stacks"))
	return ""
}

// WriteBlockProfile writes a goroutine blocking profile to the given file.
func (a *DebugAPI) WriteBlockProfile(file string) error {
	methodName := "debug_writeBlockProfile"
	a.logger.Debug(noOpMethod(methodName), "file", file)
	return ErrNotImplemented(methodName)
}

// WriteMemProfile writes an allocation profile to the given file.
func (a *DebugAPI) WriteMemProfile(file string) error {
	methodName := "debug_writeMemProfile"
	a.logger.Debug(noOpMethod(methodName), "file", file)
	return ErrNotImplemented(methodName)
}

// MutexProfile turns on mutex profiling for nsec seconds and writes profile data to file.
func (a *DebugAPI) MutexProfile(file string, nsec uint) error {
	methodName := "debug_mutexProfile"
	a.logger.Debug(noOpMethod(methodName), "file", file, "nsec", nsec)
	return ErrNotImplemented(methodName)
}

// SetMutexProfileFraction sets the rate of mutex profiling.
func (a *DebugAPI) SetMutexProfileFraction(rate int) {
	a.logger.Debug(noOpMethod("debug_setMutexProfileFraction"), "rate", rate)
}

// WriteMutexProfile writes a goroutine blocking profile to the given file.
func (a *DebugAPI) WriteMutexProfile(file string) error {
	methodName := "debug_writeMutexProfile"
	a.logger.Debug(noOpMethod(methodName), "file", file)
	return ErrNotImplemented(methodName)
}

// FreeOSMemory forces a garbage collection.
func (a *DebugAPI) FreeOSMemory() {
	a.logger.Debug(noOpMethod("debug_freeOSMemory"))
}

// SetGCPercent sets the garbage collection target percentage. It returns the previous
// setting. A negative value disables GC.
func (a *DebugAPI) SetGCPercent(v int) int {
	a.logger.Debug(noOpMethod("debug_setGCPercent"), "percent", v)
	return -1
}

// IntermediateRoots executes a block, and returns a list
// of intermediate roots: the stateroot after each transaction.
func (a *DebugAPI) IntermediateRoots(hash common.Hash, _ *evm.TraceConfig) ([]common.Hash, error) {
	a.logger.Debug("debug_intermediateRoots", "hash", hash)
	return ([]common.Hash)(nil), nil
}

// GetBadBlocks returns a list of the last 'bad blocks' that the client has seen
// on the network and returns them as a JSON list of block hashes.
func (a *DebugAPI) GetBadBlocks(ctx context.Context) ([]*getheth.BadBlockArgs, error) {
	a.logger.Debug("debug_getBadBlocks")
	return []*getheth.BadBlockArgs{}, nil
}

// GetRawBlock returns an RLP-encoded block.
func (a *DebugAPI) GetRawBlock(
	ctx context.Context,
	blockNrOrHash rpc.BlockNumberOrHash,
) (hexutil.Bytes, error) {
	fnName := "debug_getRawBlock"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

// GetRawReceipts returns an array of EIP-2718 binary-encoded receipts.
func (a *DebugAPI) GetRawReceipts(
	ctx context.Context,
	blockNrOrHash rpc.BlockNumberOrHash,
) ([]hexutil.Bytes, error) {
	fnName := "debug_getRawReceipts"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

// GetRawHeader returns an RLP-encoded block header.
func (a *DebugAPI) GetRawHeader(
	ctx context.Context,
	blockNrOrHash rpc.BlockNumberOrHash,
) (hexutil.Bytes, error) {
	fnName := "debug_getRawHeader"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

// GetRawTransaction returns the bytes of the transaction for the given hash.
func (a *DebugAPI) GetRawTransaction(
	ctx context.Context,
	hash common.Hash,
) (hexutil.Bytes, error) {
	fnName := "debug_getRawTransaction"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

// StandardTraceBadBlockToFile dumps structured logs for a bad block to file.
func (a *DebugAPI) StandardTraceBadBlockToFile(
	ctx context.Context,
	hash common.Hash,
	config *tracers.StdTraceConfig,
) ([]string, error) {
	fnName := "debug_standardTraceBadBlockToFile"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

// StandardTraceBlockToFile dumps structured logs for a block to file.
func (a *DebugAPI) StandardTraceBlockToFile(
	ctx context.Context,
	hash common.Hash,
	config *tracers.StdTraceConfig,
) ([]string, error) {
	fnName := "debug_standardTraceBlockToFile"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

// TraceBadBlock returns structured logs created during execution of a bad block.
func (a *DebugAPI) TraceBadBlock(
	ctx context.Context,
	hash common.Hash,
	config *tracers.TraceConfig,
) ([]json.RawMessage, error) {
	fnName := "debug_traceBadBlock"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

// TraceBlock returns structured logs created during execution of EVM.
func (a *DebugAPI) TraceBlock(
	ctx context.Context,
	blob hexutil.Bytes,
	config *tracers.TraceConfig,
) ([]json.RawMessage, error) {
	fnName := "debug_traceBlock"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

// TraceBlockFromFile returns structured logs created during execution from file.
func (a *DebugAPI) TraceBlockFromFile(
	ctx context.Context,
	file string,
	config *tracers.TraceConfig,
) ([]json.RawMessage, error) {
	fnName := "debug_traceBlockFromFile"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

// TraceChain returns structured logs created between two blocks.
func (a *DebugAPI) TraceChain(
	ctx context.Context,
	start, end rpc.BlockNumber,
	config *tracers.TraceConfig,
) (any, error) {
	fnName := "debug_traceChain"
	a.logger.Debug(noOpMethod(fnName))
	return nil, ErrNotImplemented(fnName)
}

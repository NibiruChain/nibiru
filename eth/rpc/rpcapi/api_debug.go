package rpcapi

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime" // #nosec G702
	"runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	pkgerrors "github.com/pkg/errors"

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
	handler *HandlerT
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
		handler: new(HandlerT),
	}
}

// HandlerT keeps track of the cpu profiler and trace execution
type HandlerT struct {
	cpuFilename   string
	cpuFile       io.WriteCloser
	mu            sync.Mutex
	traceFilename string
	traceFile     io.WriteCloser
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (a *DebugAPI) TraceTransaction(hash common.Hash, config *evm.TraceConfig) (any, error) {
	a.logger.Debug("debug_traceTransaction", "hash", hash)
	return a.backend.TraceTransaction(hash, config)
}

// TraceBlockByNumber returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (a *DebugAPI) TraceBlockByNumber(height rpc.BlockNumber, config *evm.TraceConfig) ([]*evm.TxTraceResult, error) {
	a.logger.Debug("debug_traceBlockByNumber", "height", height)
	if height == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	// Get Tendermint Block
	resBlock, err := a.backend.TendermintBlockByNumber(height)
	if err != nil {
		a.logger.Debug("get block failed", "height", height, "error", err.Error())
		return nil, err
	}

	return a.backend.TraceBlock(rpc.BlockNumber(resBlock.Block.Height), config, resBlock)
}

// TraceBlockByHash returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (a *DebugAPI) TraceBlockByHash(hash common.Hash, config *evm.TraceConfig) ([]*evm.TxTraceResult, error) {
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

// BlockProfile turns on goroutine profiling for nsec seconds and writes profile data to
// file. It uses a profile rate of 1 for most accurate information. If a different rate is
// desired, set the rate and write the profile manually.
func (a *DebugAPI) BlockProfile(file string, nsec uint) error {
	a.logger.Debug("debug_blockProfile", "file", file, "nsec", nsec)
	runtime.SetBlockProfileRate(1)
	defer runtime.SetBlockProfileRate(0)

	time.Sleep(time.Duration(nsec) * time.Second)
	return writeProfile("block", file, a.logger)
}

// CpuProfile turns on CPU profiling for nsec seconds and writes
// profile data to file.
func (a *DebugAPI) CpuProfile(file string, nsec uint) error { //nolint: golint, stylecheck, revive
	a.logger.Debug("debug_cpuProfile", "file", file, "nsec", nsec)
	if err := a.StartCPUProfile(file); err != nil {
		return err
	}
	time.Sleep(time.Duration(nsec) * time.Second)
	return a.StopCPUProfile()
}

// GcStats returns GC statistics.
func (a *DebugAPI) GcStats() *debug.GCStats {
	a.logger.Debug("debug_gcStats")
	s := new(debug.GCStats)
	debug.ReadGCStats(s)
	return s
}

// GoTrace turns on tracing for nsec seconds and writes
// trace data to file.
func (a *DebugAPI) GoTrace(file string, nsec uint) error {
	a.logger.Debug("debug_goTrace", "file", file, "nsec", nsec)
	if err := a.StartGoTrace(file); err != nil {
		return err
	}
	time.Sleep(time.Duration(nsec) * time.Second)
	return a.StopGoTrace()
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
	a.logger.Debug("debug_setBlockProfileRate", "rate", rate)
	runtime.SetBlockProfileRate(rate)
}

// Stacks returns a printed representation of the stacks of all goroutines.
func (a *DebugAPI) Stacks() string {
	a.logger.Debug("debug_stacks")
	buf := new(bytes.Buffer)
	err := pprof.Lookup("goroutine").WriteTo(buf, 2)
	if err != nil {
		a.logger.Error("Failed to create stacks", "error", err.Error())
	}
	return buf.String()
}

// StartCPUProfile turns on CPU profiling, writing to the given file.
func (a *DebugAPI) StartCPUProfile(file string) error {
	a.logger.Debug("debug_startCPUProfile", "file", file)
	a.handler.mu.Lock()
	defer a.handler.mu.Unlock()

	switch {
	case isCPUProfileConfigurationActivated(a.ctx):
		a.logger.Debug("CPU profiling already in progress using the configuration file")
		return errors.New("CPU profiling already in progress using the configuration file")
	case a.handler.cpuFile != nil:
		a.logger.Debug("CPU profiling already in progress")
		return errors.New("CPU profiling already in progress")
	default:
		fp, err := ExpandHome(file)
		if err != nil {
			a.logger.Debug("failed to get filepath for the CPU profile file", "error", err.Error())
			return err
		}
		f, err := os.Create(fp)
		if err != nil {
			a.logger.Debug("failed to create CPU profile file", "error", err.Error())
			return err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			a.logger.Debug("cpu profiling already in use", "error", err.Error())
			if err := f.Close(); err != nil {
				a.logger.Debug("failed to close cpu profile file")
				return fmt.Errorf("failed to close cpu profile file: %w", err)
			}
			return err
		}

		a.logger.Info("CPU profiling started", "profile", file)
		a.handler.cpuFile = f
		a.handler.cpuFilename = file
		return nil
	}
}

// StopCPUProfile stops an ongoing CPU profile.
func (a *DebugAPI) StopCPUProfile() error {
	a.logger.Debug("debug_stopCPUProfile")
	a.handler.mu.Lock()
	defer a.handler.mu.Unlock()

	switch {
	case isCPUProfileConfigurationActivated(a.ctx):
		a.logger.Debug("CPU profiling already in progress using the configuration file")
		return errors.New("CPU profiling already in progress using the configuration file")
	case a.handler.cpuFile != nil:
		a.logger.Info("Done writing CPU profile", "profile", a.handler.cpuFilename)
		pprof.StopCPUProfile()
		if err := a.handler.cpuFile.Close(); err != nil {
			a.logger.Debug("failed to close cpu file")
			return fmt.Errorf("failed to close cpu file: %w", err)
		}
		a.handler.cpuFile = nil
		a.handler.cpuFilename = ""
		return nil
	default:
		a.logger.Debug("CPU profiling not in progress")
		return errors.New("CPU profiling not in progress")
	}
}

// WriteBlockProfile writes a goroutine blocking profile to the given file.
func (a *DebugAPI) WriteBlockProfile(file string) error {
	a.logger.Debug("debug_writeBlockProfile", "file", file)
	return writeProfile("block", file, a.logger)
}

// WriteMemProfile writes an allocation profile to the given file.
// Note that the profiling rate cannot be set through the API,
// it must be set on the command line.
func (a *DebugAPI) WriteMemProfile(file string) error {
	a.logger.Debug("debug_writeMemProfile", "file", file)
	return writeProfile("heap", file, a.logger)
}

// MutexProfile turns on mutex profiling for nsec seconds and writes profile data to file.
// It uses a profile rate of 1 for most accurate information. If a different rate is
// desired, set the rate and write the profile manually.
func (a *DebugAPI) MutexProfile(file string, nsec uint) error {
	a.logger.Debug("debug_mutexProfile", "file", file, "nsec", nsec)
	runtime.SetMutexProfileFraction(1)
	time.Sleep(time.Duration(nsec) * time.Second)
	defer runtime.SetMutexProfileFraction(0)
	return writeProfile("mutex", file, a.logger)
}

// SetMutexProfileFraction sets the rate of mutex profiling.
func (a *DebugAPI) SetMutexProfileFraction(rate int) {
	a.logger.Debug("debug_setMutexProfileFraction", "rate", rate)
	runtime.SetMutexProfileFraction(rate)
}

// WriteMutexProfile writes a goroutine blocking profile to the given file.
func (a *DebugAPI) WriteMutexProfile(file string) error {
	a.logger.Debug("debug_writeMutexProfile", "file", file)
	return writeProfile("mutex", file, a.logger)
}

// FreeOSMemory forces a garbage collection.
func (a *DebugAPI) FreeOSMemory() {
	a.logger.Debug("debug_freeOSMemory")
	debug.FreeOSMemory()
}

// SetGCPercent sets the garbage collection target percentage. It returns the previous
// setting. A negative value disables GC.
func (a *DebugAPI) SetGCPercent(v int) int {
	a.logger.Debug("debug_setGCPercent", "percent", v)
	return debug.SetGCPercent(v)
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

func ErrNotImplemented(method string) error {
	return fmt.Errorf("method is not implemented: %v", method)
}

// GetRawBlock returns an RLP-encoded block
func (a *DebugAPI) GetRawBlock(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	fnName := "debug_getRawBlock"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// GetRawReceipts returns an array of EIP-2718 binary-encoded receipts
func (a *DebugAPI) GetRawReceipts(
	ctx context.Context,
	blockNrOrHash rpc.BlockNumberOrHash,
) ([]hexutil.Bytes, error) {
	fnName := "debug_getRawReceipts"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// GetRawHeader returns an RLP-encoded block header
func (a *DebugAPI) GetRawHeader(
	ctx context.Context,
	blockNrOrHash rpc.BlockNumberOrHash,
) (hexutil.Bytes, error) {
	fnName := "debug_getRawHeader"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// GetRawTransaction returns the bytes of the transaction for the given hash.
func (a *DebugAPI) GetRawTransaction(
	ctx context.Context,
	hash common.Hash,
) (hexutil.Bytes, error) {
	fnName := "debug_getRawTransaction"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// StandardTraceBadBlockToFile dumps the structured logs created during the
// execution of EVM against a block pulled from the pool of bad ones to the
// local file system and returns a list of files to the caller.
func (a *DebugAPI) StandardTraceBadBlockToFile(
	ctx context.Context,
	hash common.Hash,
	config *tracers.StdTraceConfig,
) ([]string, error) {
	fnName := "debug_standardTraceBadBlockToFile"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// StandardTraceBlockToFile dumps the structured logs created during the
// execution of EVM to the local file system and returns a list of files
// to the caller.
func (a *DebugAPI) StandardTraceBlockToFile(
	ctx context.Context,
	hash common.Hash,
	config *tracers.StdTraceConfig,
) ([]string, error) {
	fnName := "debug_standardTraceBlockToFile"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// TraceBadBlock returns the structured logs created during the execution of
// EVM against a block pulled from the pool of bad ones and returns them as a JSON
// object.
func (a *DebugAPI) TraceBadBlock(
	ctx context.Context,
	hash common.Hash,
	config *tracers.TraceConfig,
) ([]json.RawMessage, error) {
	fnName := "debug_traceBadBlock"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// TraceBlock returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (a *DebugAPI) TraceBlock(
	ctx context.Context,
	blob hexutil.Bytes,
	config *tracers.TraceConfig,
) ([]json.RawMessage, error) {
	fnName := "debug_traceBlock"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// TraceBlockFromFile returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (a *DebugAPI) TraceBlockFromFile(
	ctx context.Context,
	file string,
	config *tracers.TraceConfig,
) ([]json.RawMessage, error) {
	fnName := "debug_traceBlockFromFile"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// TraceChain returns the structured logs created during the execution of EVM
// between two blocks (excluding start) and returns them as a JSON object.
func (a *DebugAPI) TraceChain(
	ctx context.Context,
	start, end rpc.BlockNumber,
	config *tracers.TraceConfig,
) (subscription any, err error) { // Fetch the block interval that we want to trace
	fnName := "debug_traceChain"
	a.logger.Debug(fnName)
	return nil, ErrNotImplemented(fnName)
}

// StartGoTrace turns on tracing, writing to the given file.
func (a *DebugAPI) StartGoTrace(file string) error {
	a.logger.Debug("debug_startGoTrace", "file", file)
	a.handler.mu.Lock()
	defer a.handler.mu.Unlock()

	if a.handler.traceFile != nil {
		a.logger.Debug("trace already in progress")
		return errors.New("trace already in progress")
	}
	fp, err := ExpandHome(file)
	if err != nil {
		a.logger.Debug("failed to get filepath for the CPU profile file", "error", err.Error())
		return err
	}
	f, err := os.Create(fp)
	if err != nil {
		a.logger.Debug("failed to create go trace file", "error", err.Error())
		return err
	}
	if err := trace.Start(f); err != nil {
		a.logger.Debug("Go tracing already started", "error", err.Error())
		if err := f.Close(); err != nil {
			a.logger.Debug("failed to close trace file")
			return pkgerrors.Wrap(err, "failed to close trace file")
		}

		return err
	}
	a.handler.traceFile = f
	a.handler.traceFilename = file
	a.logger.Info("Go tracing started", "dump", a.handler.traceFilename)
	return nil
}

// StopGoTrace stops an ongoing trace.
func (a *DebugAPI) StopGoTrace() error {
	a.logger.Debug("debug_stopGoTrace")
	a.handler.mu.Lock()
	defer a.handler.mu.Unlock()

	trace.Stop()
	if a.handler.traceFile == nil {
		a.logger.Debug("trace not in progress")
		return errors.New("trace not in progress")
	}
	a.logger.Info("Done writing Go trace", "dump", a.handler.traceFilename)
	if err := a.handler.traceFile.Close(); err != nil {
		a.logger.Debug("failed to close trace file")
		return pkgerrors.Wrap(err, "failed to close trace file")
	}
	a.handler.traceFile = nil
	a.handler.traceFilename = ""
	return nil
}

// isCPUProfileConfigurationActivated: Checks if the "cpu-profile" flag was set
func isCPUProfileConfigurationActivated(ctx *server.Context) bool {
	// TODO: use same constants as server/start.go
	// constant declared in start.go cannot be imported (cyclical dependency)
	const flagCPUProfile = "cpu-profile"
	if cpuProfile := ctx.Viper.GetString(flagCPUProfile); cpuProfile != "" {
		return true
	}
	return false
}

// ExpandHome expands home directory in file paths.
// ~someuser/tmp will not be expanded.
func ExpandHome(p string) (string, error) {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		usr, err := user.Current()
		if err != nil {
			return p, err
		}
		home := usr.HomeDir
		p = home + p[1:]
	}
	return filepath.Clean(p), nil
}

// writeProfile writes the data to a file
func writeProfile(name, file string, log log.Logger) error {
	p := pprof.Lookup(name)
	log.Info("Writing profile records", "count", p.Count(), "type", name, "dump", file)
	fp, err := ExpandHome(file)
	if err != nil {
		return err
	}
	f, err := os.Create(fp)
	if err != nil {
		return err
	}

	if err := p.WriteTo(f, 0); err != nil {
		if err := f.Close(); err != nil {
			return err
		}
		return err
	}

	return f.Close()
}

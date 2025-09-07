// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"os"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	tracersnative "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/params"
)

const (
	TracerAccessList = "access_list"
	TracerJSON       = "json"
	TracerStruct     = "struct"
	TracerMarkdown   = "markdown"
)

// NewTracer creates a new Logger tracer to collect execution traces from an
// EVM transaction.
func NewTracer(
	tracer string,
	msg core.Message,
	cfg *params.ChainConfig,
	height int64,
) *tracing.Hooks {
	// TODO: feat(evm-vmtracer): enable additional log configuration
	logCfg := &logger.Config{
		Debug: false,
	}

	switch tracer {
	case TracerAccessList:
		precompileAddrs := PRECOMPILE_ADDRS
		return logger.NewAccessListTracer(
			msg.AccessList,
			msg.From,
			*msg.To,
			precompileAddrs,
		).Hooks()
	case TracerJSON:
		return logger.NewJSONLogger(logCfg, os.Stdout)
	case TracerMarkdown:
		return logger.NewMarkdownLogger(logCfg, os.Stdout).Hooks()
	case TracerStruct:
		return NewDefaultTracer().Hooks
	default:
		// The no-op tracer, `return NewNoOpTracer().Hooks` is meant for testing
		// in geth, not production.
		return NewDefaultTracer().Hooks
	}
}

func NewDefaultTracer() *tracers.Tracer {
	logCfg := &logger.Config{
		Debug: false,
	}
	defaultLogger := logger.NewStructLogger(logCfg)
	return &tracers.Tracer{
		Hooks:     defaultLogger.Hooks(),
		GetResult: defaultLogger.GetResult,
		Stop:      defaultLogger.Stop,
	}
}

// TxTraceResult is the result of a single transaction trace during a block trace.
type TxTraceResult struct {
	Result any    `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// NewNoOpTracer creates a no-op vm.Tracer
func NewNoOpTracer() (tracer *tracers.Tracer) {
	// This function cannot error, so we ignore the second return value
	tracer, _ = tracersnative.NewNoopTracer(nil, nil, nil)
	return tracer
}

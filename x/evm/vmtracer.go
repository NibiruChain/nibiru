// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
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
func NewTracer(tracer string, msg core.Message, cfg *params.ChainConfig, height int64) *tracing.Hooks {
	// TODO: enable additional log configuration
	logCfg := &logger.Config{
		Debug: true,
	}

	// FIXME: inconsistent logging between stdout and stderr
	switch tracer {
	case TracerAccessList:
		precompiles := vm.ActivePrecompiles(cfg.Rules(big.NewInt(height), cfg.MergeNetsplitBlock != nil, 0))
		return logger.NewAccessListTracer(msg.AccessList, msg.From, *msg.To, precompiles).Hooks()
	case TracerJSON:
		return logger.NewJSONLogger(logCfg, os.Stderr)
	case TracerMarkdown:
		return logger.NewMarkdownLogger(logCfg, os.Stdout).Hooks()
	case TracerStruct:
		return logger.NewStructLogger(logCfg).Hooks()
	default:
		noopTracer, _ := tracers.DefaultDirectory.New("noopTracer", nil, nil)
		return noopTracer.Hooks
	}
}

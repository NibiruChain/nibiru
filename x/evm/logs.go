// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"errors"
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/eth"
)

// NewTransactionLogs creates a new NewTransactionLogs instance.
func NewTransactionLogs(hash gethcommon.Hash, logs []*Log) TransactionLogs {
	return TransactionLogs{
		Hash: hash.String(),
		Logs: logs,
	}
}

// NewTransactionLogsFromEth creates a new NewTransactionLogs instance using []*ethtypes.Log.
func NewTransactionLogsFromEth(hash gethcommon.Hash, ethlogs []*gethcore.Log) TransactionLogs {
	return TransactionLogs{
		Hash: hash.String(),
		Logs: NewLogsFromEth(ethlogs),
	}
}

// Validate performs a basic validation of a GenesisAccount fields.
func (tx *TransactionLogs) Validate() error {
	if eth.IsEmptyHash(tx.Hash) {
		return fmt.Errorf("hash cannot be the empty %s", tx.Hash)
	}

	for i, log := range tx.Logs {
		if log == nil {
			return fmt.Errorf("log %d cannot be nil", i)
		}
		if err := log.Validate(); err != nil {
			return fmt.Errorf("invalid log %d: %w", i, err)
		}
		if log.TxHash != tx.Hash {
			return fmt.Errorf("log tx hash mismatch (%s ≠ %s)", log.TxHash, tx.Hash)
		}
	}
	return nil
}

// EthLogs returns the Ethereum type Logs from the Transaction Logs.
func (tx *TransactionLogs) EthLogs() []*gethcore.Log {
	return LogsToEthereum(tx.Logs)
}

// Validate performs a basic validation of an ethereum Log fields.
func (log *Log) Validate() error {
	if err := eth.ValidateAddress(log.Address); err != nil {
		return fmt.Errorf("invalid log address %w", err)
	}
	if eth.IsEmptyHash(log.BlockHash) {
		return fmt.Errorf("block hash cannot be the empty %s", log.BlockHash)
	}
	if log.BlockNumber == 0 {
		return errors.New("block number cannot be zero")
	}
	if eth.IsEmptyHash(log.TxHash) {
		return fmt.Errorf("tx hash cannot be the empty %s", log.TxHash)
	}
	return nil
}

// ToEthereum returns the Ethereum type Log from a Ethermint proto compatible Log.
func (log *Log) ToEthereum() *gethcore.Log {
	topics := make([]gethcommon.Hash, len(log.Topics))
	for i, topic := range log.Topics {
		topics[i] = gethcommon.HexToHash(topic)
	}

	return &gethcore.Log{
		Address:     gethcommon.HexToAddress(log.Address),
		Topics:      topics,
		Data:        log.Data,
		BlockNumber: log.BlockNumber,
		TxHash:      gethcommon.HexToHash(log.TxHash),
		TxIndex:     uint(log.TxIndex),
		Index:       uint(log.Index),
		BlockHash:   gethcommon.HexToHash(log.BlockHash),
		Removed:     log.Removed,
	}
}

func NewLogsFromEth(ethlogs []*gethcore.Log) []*Log {
	var logs []*Log //nolint: prealloc
	for _, ethlog := range ethlogs {
		logs = append(logs, NewLogFromEth(ethlog))
	}

	return logs
}

// LogsToEthereum casts the Proto Logs to a slice of Ethereum Logs.
func LogsToEthereum(logs []*Log) []*gethcore.Log {
	var ethLogs []*gethcore.Log //nolint: prealloc
	for i := range logs {
		ethLogs = append(ethLogs, logs[i].ToEthereum())
	}
	return ethLogs
}

// NewLogFromEth creates a new Log instance from an Ethereum type Log.
func NewLogFromEth(log *gethcore.Log) *Log {
	topics := make([]string, len(log.Topics))
	for i, topic := range log.Topics {
		topics[i] = topic.String()
	}

	return &Log{
		Address:     log.Address.String(),
		Topics:      topics,
		Data:        log.Data,
		BlockNumber: log.BlockNumber,
		TxHash:      log.TxHash.String(),
		TxIndex:     uint64(log.TxIndex),
		Index:       uint64(log.Index),
		BlockHash:   log.BlockHash.String(),
		Removed:     log.Removed,
	}
}

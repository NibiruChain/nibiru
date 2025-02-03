// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"encoding/json"
	"fmt"
	"strconv"

	"cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

// Evm module events
const (
	// proto.MessageName(new(evm.EventBlockBloom))
	TypeUrlEventBlockBloom = "eth.evm.v1.EventBlockBloom"

	// proto.MessageName(new(evm.EventTxLog))
	TypeUrlEventTxLog = "eth.evm.v1.EventTxLog"

	// proto.MessageName(new(evm.TypeUrlEventEthereumTx))
	TypeUrlEventEthereumTx = "eth.evm.v1.EventEthereumTx"

	// Untyped events and attribuges

	// Used in non-typed event "message"
	MessageEventAttrTxType = "tx_type"

	// Used in non-typed event "pending_ethereum_tx"
	PendingEthereumTxEvent            = "pending_ethereum_tx"
	PendingEthereumTxEventAttrEthHash = "eth_hash"
	PendingEthereumTxEventAttrIndex   = "index"
)

func EventTxLogFromABCIEvent(event abci.Event) (*EventTxLog, error) {
	typedProtoEvent, err := sdk.ParseTypedEvent(event)

	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", TypeUrlEventTxLog)
	}
	typedEvent, ok := (typedProtoEvent).(*EventTxLog)
	if !ok {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", TypeUrlEventTxLog)
	}
	if typedEvent.Logs == nil {
		legacyLogs := TryParseLegacyTxLogsFromABCIEvent(event)
		if legacyLogs != nil {
			typedEvent.Logs = legacyLogs
		}
	}
	return typedEvent, nil
}

// TryParseLegacyTxLogsFromABCIEvent is a fallback for parsing logs in the legacy format.
// See: https://github.com/NibiruChain/nibiru/pull/2188
func TryParseLegacyTxLogsFromABCIEvent(event abci.Event) []Log {
	var legacyLogs []Log
	for _, attr := range event.Attributes {
		if attr.Key != "tx_logs" {
			continue
		}
		var strLogs []string
		if err := json.Unmarshal([]byte(attr.Value), &strLogs); err != nil {
			return nil
		}
		for _, strLog := range strLogs {
			var log Log
			if err := json.Unmarshal([]byte(strLog), &log); err != nil {
				return nil
			}
			legacyLogs = append(legacyLogs, log)
		}
	}
	return legacyLogs
}

func EventBlockBloomFromABCIEvent(event abci.Event) (*EventBlockBloom, error) {
	typedProtoEvent, err := sdk.ParseTypedEvent(event)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", TypeUrlEventBlockBloom)
	}
	typedEvent, ok := (typedProtoEvent).(*EventBlockBloom)
	if !ok {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", TypeUrlEventBlockBloom)
	}
	return typedEvent, nil
}

func EventEthereumTxFromABCIEvent(event abci.Event) (*EventEthereumTx, error) {
	typedProtoEvent, err := sdk.ParseTypedEvent(event)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", TypeUrlEventEthereumTx)
	}
	typedEvent, ok := (typedProtoEvent).(*EventEthereumTx)
	if !ok {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", TypeUrlEventEthereumTx)
	}
	return typedEvent, nil
}

func GetEthHashAndIndexFromPendingEthereumTxEvent(event abci.Event) (gethcommon.Hash, int32, error) {
	ethHash := gethcommon.Hash{}
	txIndex := int32(-1)

	for _, attr := range event.Attributes {
		if attr.Key == PendingEthereumTxEventAttrEthHash {
			ethHash = gethcommon.HexToHash(attr.Value)
		}
		if attr.Key == PendingEthereumTxEventAttrIndex {
			parsedIndex, err := strconv.ParseInt(attr.Value, 10, 32)
			if err != nil {
				return ethHash, -1, fmt.Errorf(
					"failed to parse tx index from pending_ethereum_tx event, %s", attr.Value,
				)
			}
			txIndex = int32(parsedIndex)
		}
	}
	if txIndex == -1 {
		return ethHash, -1, fmt.Errorf("tx index not found in pending_ethereum_tx")
	}
	if ethHash.String() == "" {
		return ethHash, -1, fmt.Errorf("eth hash not found in pending_ethereum_tx")
	}
	return ethHash, txIndex, nil
}

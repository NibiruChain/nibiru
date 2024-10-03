// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
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
	typeUrl := TypeUrlEventTxLog
	typedProtoEvent, err := sdk.ParseTypedEvent(event)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", typeUrl)
	}
	typedEvent, ok := (typedProtoEvent).(*EventTxLog)
	if !ok {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", typeUrl)
	}
	return typedEvent, nil
}

func EventBlockBloomFromABCIEvent(event abci.Event) (*EventBlockBloom, error) {
	typeUrl := TypeUrlEventBlockBloom
	typedProtoEvent, err := sdk.ParseTypedEvent(event)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", typeUrl)
	}
	typedEvent, ok := (typedProtoEvent).(*EventBlockBloom)
	if !ok {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", typeUrl)
	}
	return typedEvent, nil
}

func EventEthereumTxFromABCIEvent(event abci.Event) (*EventEthereumTx, error) {
	typeUrl := TypeUrlEventEthereumTx
	typedProtoEvent, err := sdk.ParseTypedEvent(event)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", typeUrl)
	}
	typedEvent, ok := (typedProtoEvent).(*EventEthereumTx)
	if !ok {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", typeUrl)
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

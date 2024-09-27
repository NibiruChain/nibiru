// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Evm module events
const (
	// proto.MessageName(new(evm.EventBlockBloom))
	TypeUrlEventBlockBloom = "eth.evm.v1.EventBlockBloom"

	// proto.MessageName(new(evm.EventTxLog))
	TypeUrlEventTxLog = "eth.evm.v1.EventTxLog"

	// proto.MessageName(new(evm.EventPendingEthereumTx))
	TypeUrlEventPendingEthereumTx = "eth.evm.v1.EventPendingEthereumTx"

	// proto.MessageName(new(evm.EventTxLog))
	TypeUrlEventEthereumTx = "eth.evm.v1.EventEthereumTx"
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

func EventPendingEthereumTxFromABCIEvent(event abci.Event) (*EventPendingEthereumTx, error) {
	typeUrl := TypeUrlEventPendingEthereumTx
	typedProtoEvent, err := sdk.ParseTypedEvent(event)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse event of type %s", typeUrl)
	}
	typedEvent, ok := (typedProtoEvent).(*EventPendingEthereumTx)
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

// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Evm module events
const (
	EventTypeEthereumTx = TypeMsgEthereumTx
	EventTypeTxLog      = "tx_log"

	// proto.MessageName(new(evm.EventBlockBloom))
	TypeUrlEventBlockBloom = "eth.evm.v1.EventBlockBloom"

	// proto.MessageName(new(evm.EventTxLog))
	TypeUrlEventTxLog = "eth.evm.v1.EventTxLog"

	AttributeKeyRecipient      = "recipient"
	AttributeKeyTxHash         = "txHash"
	AttributeKeyEthereumTxHash = "ethereumTxHash"
	AttributeKeyTxIndex        = "txIndex"
	AttributeKeyTxGasUsed      = "txGasUsed"
	AttributeKeyTxType         = "txType"
	AttributeKeyTxLog          = "txLog"
	// tx failed in eth vm execution
	AttributeKeyEthereumTxFailed = "ethereumTxFailed"
	// JSON name of EventBlockBloom.Bloom
	AttributeKeyEthereumBloom = "bloom"
)

func (e *EventTxLog) FromABCIEvent(event abci.Event) (*EventTxLog, error) {
	typeUrl := TypeUrlEventTxLog
	typedProtoEvent, err := sdk.ParseTypedEvent(event)
	if err != nil {
		return nil, errors.Wrapf(
			err, "EventTxLog.FromABCIEvent failed to parse event of type %s", typeUrl)
	}
	typedEvent, ok := (typedProtoEvent).(*EventTxLog)
	if !ok {
		return nil, errors.Wrapf(
			err, "EventTxLog.FromABCIEvent failed to parse event of type %s", typeUrl)
	}
	return typedEvent, nil
}

func (e *EventBlockBloom) FromABCIEvent(event abci.Event) (*EventBlockBloom, error) {
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

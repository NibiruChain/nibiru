// Copyright (c) 2023-2024 Nibi, Inc.
package evm

// Evm module events
const (
	EventTypeEthereumTx = TypeMsgEthereumTx

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

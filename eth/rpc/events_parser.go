// Copyright (c) 2023-2024 Nibi, Inc.
package rpc

import (
	"errors"
	"fmt"
	"strconv"

	abci "github.com/cometbft/cometbft/abci/types"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// ParsedTx is eth tx info parsed from ABCI events. Each `ParsedTx` corresponds
// to one eth tx msg ([evm.MsgEthereumTx]).
type ParsedTx struct {
	MsgIndex int

	// the following fields are parsed from events
	EthHash gethcommon.Hash

	EthTxIndex int32 // -1 means uninitialized
	GasUsed    uint64
	Failed     bool
}

// ParsedTxs is the tx infos parsed from eth tx events.
type ParsedTxs struct {
	// one item per message
	Txs []ParsedTx
	// map tx hash to msg index
	TxHashes map[gethcommon.Hash]int
}

// ParseTxResult parses eth tx info from the ABCI events of Eth tx msgs
func ParseTxResult(result *abci.ResponseDeliverTx, tx sdk.Tx) (*ParsedTxs, error) {
	eventTypePendingEthereumTx := evm.PendingEthereumTxEvent
	eventTypeEthereumTx := proto.MessageName((*evm.EventEthereumTx)(nil))

	// Parsed txs is the structure being populated from the events
	// So far (until we allow ethereum_txs as cosmos tx messages) it'll have single tx
	parsedTxs := &ParsedTxs{
		Txs:      make([]ParsedTx, 0),
		TxHashes: make(map[gethcommon.Hash]int),
	}

	// msgIndex counts only ethereum tx messages.
	msgIndex := -1
	for _, event := range result.Events {
		// Pending tx event could be single if tx didn't succeed
		if event.Type == eventTypePendingEthereumTx {
			msgIndex++
			ethHash, txIndex, err := evm.GetEthHashAndIndexFromPendingEthereumTxEvent(event)
			if err != nil {
				return nil, err
			}
			pendingTx := ParsedTx{
				MsgIndex:   msgIndex,
				EthTxIndex: txIndex,
				EthHash:    ethHash,
			}
			parsedTxs.Txs = append(parsedTxs.Txs, pendingTx)
			parsedTxs.TxHashes[ethHash] = msgIndex
		} else if event.Type == eventTypeEthereumTx { // Full event replaces the pending tx
			eventEthereumTx, err := evm.EventEthereumTxFromABCIEvent(event)
			if err != nil {
				return nil, err
			}
			ethTxIndexFromEvent, err := strconv.ParseUint(eventEthereumTx.Index, 10, 31)
			if err != nil {
				return nil, fmt.Errorf("failed to parse EthTxIndex from event: %w", err)
			}
			gasUsed, err := strconv.ParseUint(eventEthereumTx.GasUsed, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GasUsed from event: %w", err)
			}
			committedTx := ParsedTx{
				MsgIndex:   msgIndex,
				EthTxIndex: int32(ethTxIndexFromEvent),
				EthHash:    gethcommon.HexToHash(eventEthereumTx.EthHash),
				GasUsed:    gasUsed,
				Failed:     len(eventEthereumTx.EthTxFailed) > 0,
			}
			// replace pending tx with committed tx
			if msgIndex >= 0 && len(parsedTxs.Txs) == msgIndex+1 {
				parsedTxs.Txs[msgIndex] = committedTx
			} else {
				// EventEthereumTx without EventPendingEthereumTx
				return nil, errors.New("EventEthereumTx without pending_ethereum_tx event")
			}
		}
	}

	// this could only happen if tx exceeds block gas limit
	if result.Code != 0 && tx != nil {
		for i := 0; i < len(parsedTxs.Txs); i++ {
			parsedTxs.Txs[i].Failed = true

			// replace gasUsed with gasLimit because that's what's actually deducted.
			msgEthereumTx, ok := tx.GetMsgs()[i].(*evm.MsgEthereumTx)
			if !ok {
				return nil, fmt.Errorf("unexpected message type at index %d", i)
			}
			gasLimit := msgEthereumTx.GetGas()
			parsedTxs.Txs[i].GasUsed = gasLimit
		}
	}
	return parsedTxs, nil
}

// ParseTxIndexerResult parse tm tx result to a format compatible with the custom tx indexer.
func ParseTxIndexerResult(txResult *tmrpctypes.ResultTx, tx sdk.Tx, getter func(*ParsedTxs) *ParsedTx) (*eth.TxResult, error) {
	txs, err := ParseTxResult(&txResult.TxResult, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tx events: block %d, index %d, %v", txResult.Height, txResult.Index, err)
	}

	parsedTx := getter(txs)
	if parsedTx == nil {
		return nil, fmt.Errorf("ethereum tx not found in msgs: block %d, index %d", txResult.Height, txResult.Index)
	}
	index := uint32(parsedTx.MsgIndex) // #nosec G701
	return &eth.TxResult{
		Height:            txResult.Height,
		TxIndex:           txResult.Index,
		MsgIndex:          index,
		EthTxIndex:        parsedTx.EthTxIndex,
		Failed:            parsedTx.Failed,
		GasUsed:           parsedTx.GasUsed,
		CumulativeGasUsed: txs.AccumulativeGasUsed(parsedTx.MsgIndex),
	}, nil
}

// GetTxByHash find ParsedTx by tx hash, returns nil if not exists.
func (p *ParsedTxs) GetTxByHash(hash gethcommon.Hash) *ParsedTx {
	if idx, ok := p.TxHashes[hash]; ok {
		return &p.Txs[idx]
	}
	return nil
}

// GetTxByMsgIndex returns ParsedTx by msg index
func (p *ParsedTxs) GetTxByMsgIndex(i int) *ParsedTx {
	if i < 0 || i >= len(p.Txs) {
		return nil
	}
	return &p.Txs[i]
}

// GetTxByTxIndex returns ParsedTx by tx index
func (p *ParsedTxs) GetTxByTxIndex(txIndex int) *ParsedTx {
	if len(p.Txs) == 0 {
		return nil
	}
	// assuming the `EthTxIndex` increase continuously,
	// convert TxIndex to MsgIndex by subtract the begin TxIndex.
	msgIndex := txIndex - int(p.Txs[0].EthTxIndex)
	// GetTxByMsgIndex will check the bound
	return p.GetTxByMsgIndex(msgIndex)
}

// AccumulativeGasUsed calculates the accumulated gas used within the batch of txs
func (p *ParsedTxs) AccumulativeGasUsed(msgIndex int) (result uint64) {
	if msgIndex < 0 || msgIndex >= len(p.Txs) {
		return 0
	}
	for i := 0; i <= msgIndex; i++ {
		result += p.Txs[i].GasUsed
	}
	return result
}

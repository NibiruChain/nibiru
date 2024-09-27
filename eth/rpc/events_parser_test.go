package rpc

import (
	"math/big"
	"strconv"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func TestParseTxResult(t *testing.T) {
	txHashOne := gethcommon.BigToHash(big.NewInt(1))
	txHashTwo := gethcommon.BigToHash(big.NewInt(2))

	type TestCase struct {
		name       string
		txResp     abci.ResponseDeliverTx
		wantEthTxs []*ParsedTx
		wantErr    bool
	}

	testCases := []TestCase{
		{
			name: "happy: valid single pending_ethereum_tx event",
			txResp: abci.ResponseDeliverTx{
				Events: []abci.Event{
					pendingEthereumTxEvent(txHashOne.Hex(), 0),
				},
			},
			wantEthTxs: []*ParsedTx{
				{
					MsgIndex:   0,
					EthHash:    txHashOne,
					EthTxIndex: 0,
					GasUsed:    0,
					Failed:     false,
				},
			},
		},
		{
			name: "happy: two valid pending_ethereum_tx events",
			txResp: abci.ResponseDeliverTx{
				Events: []abci.Event{
					pendingEthereumTxEvent(txHashOne.Hex(), 0),
					pendingEthereumTxEvent(txHashTwo.Hex(), 1),
				},
			},
			wantEthTxs: []*ParsedTx{
				{
					MsgIndex:   0,
					EthHash:    txHashOne,
					EthTxIndex: 0,
					GasUsed:    0,
					Failed:     false,
				},
				{
					MsgIndex:   1,
					EthHash:    txHashTwo,
					EthTxIndex: 1,
					Failed:     false,
				},
			},
		},
		{
			name: "happy: one pending_ethereum_tx and one EventEthereumTx",
			txResp: abci.ResponseDeliverTx{
				Events: []abci.Event{
					pendingEthereumTxEvent(txHashOne.Hex(), 0),
					ethereumTxEvent(txHashOne.Hex(), 0, 21000, false),
				},
			},
			wantEthTxs: []*ParsedTx{
				{
					MsgIndex:   0,
					EthHash:    txHashOne,
					EthTxIndex: 0,
					GasUsed:    21000,
					Failed:     false,
				},
			},
		},
		{
			name: "happy: two pending_ethereum_tx and one EventEthereumTx",
			txResp: abci.ResponseDeliverTx{
				Events: []abci.Event{
					pendingEthereumTxEvent(txHashOne.Hex(), 0),
					pendingEthereumTxEvent(txHashTwo.Hex(), 1),
					ethereumTxEvent(txHashTwo.Hex(), 1, 21000, false),
				},
			},
			wantEthTxs: []*ParsedTx{
				{
					MsgIndex:   0,
					EthHash:    txHashOne,
					EthTxIndex: 0,
					GasUsed:    0,
					Failed:     false,
				},
				{
					MsgIndex:   1,
					EthHash:    txHashTwo,
					EthTxIndex: 1,
					GasUsed:    21000,
					Failed:     false,
				},
			},
		},
		{
			name: "happy: one pending_ethereum_tx and one EventEthereumTx failed",
			txResp: abci.ResponseDeliverTx{
				Events: []abci.Event{
					pendingEthereumTxEvent(txHashOne.Hex(), 0),
					ethereumTxEvent(txHashOne.Hex(), 0, 21000, true),
				},
			},
			wantEthTxs: []*ParsedTx{
				{
					MsgIndex:   0,
					EthHash:    txHashOne,
					EthTxIndex: 0,
					GasUsed:    21000,
					Failed:     true,
				},
			},
		},
		{
			name: "sad: EventEthereumTx without pending_ethereum_tx",
			txResp: abci.ResponseDeliverTx{
				Events: []abci.Event{
					ethereumTxEvent(txHashOne.Hex(), 0, 21000, false),
				},
			},
			wantEthTxs: nil,
			wantErr:    true,
		},
		{
			name: "sad: no events",
			txResp: abci.ResponseDeliverTx{
				Events: []abci.Event{},
			},
			wantEthTxs: []*ParsedTx{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseTxResult(&tc.txResp, nil) //#nosec G601 -- fine for tests
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			for msgIndex, expTx := range tc.wantEthTxs {
				require.Equal(t, expTx, parsed.GetTxByMsgIndex(msgIndex))
				require.Equal(t, expTx, parsed.GetTxByHash(expTx.EthHash))
				require.Equal(t, expTx, parsed.GetTxByTxIndex(int(expTx.EthTxIndex)))
				require.Equal(t, expTx.GasUsed, parsed.GetTxByHash(expTx.EthHash).GasUsed)
				require.Equal(t, expTx.Failed, parsed.GetTxByHash(expTx.EthHash).Failed)
			}
			// non-exists tx hash
			require.Nil(t, parsed.GetTxByHash(gethcommon.Hash{}))
			// out of range
			require.Nil(t, parsed.GetTxByMsgIndex(len(tc.wantEthTxs)))
			require.Nil(t, parsed.GetTxByTxIndex(99999999))
		})
	}
}

func pendingEthereumTxEvent(txHash string, txIndex int) abci.Event {
	return abci.Event{
		Type: evm.PendingEthereumTxEvent,
		Attributes: []abci.EventAttribute{
			{Key: evm.PendingEthereumTxEventAttrEthHash, Value: txHash},
			{Key: evm.PendingEthereumTxEventTxAttrIndex, Value: strconv.Itoa(txIndex)},
		},
	}
}

func ethereumTxEvent(txHash string, txIndex int, gasUsed int, failed bool) abci.Event {
	failure := ""
	if failed {
		failure = "failed"
	}
	event, err := sdk.TypedEventToEvent(
		&evm.EventEthereumTx{
			EthHash:     txHash,
			Index:       strconv.Itoa(txIndex),
			GasUsed:     strconv.Itoa(gasUsed),
			EthTxFailed: failure,
		},
	)
	if err != nil {
		panic(err)
	}
	return (abci.Event)(event)
}

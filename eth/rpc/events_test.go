package rpc

import (
	"math/big"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/evm"
)

func TestParseTxResult(t *testing.T) {
	address := "0x57f96e6B86CdeFdB3d412547816a82E3E0EbF9D2"
	txHash := common.BigToHash(big.NewInt(1))
	txHash2 := common.BigToHash(big.NewInt(2))

	type TestCase struct {
		name       string
		txResp     abci.ResponseDeliverTx
		wantEthTxs []*ParsedTx // expected parse result, nil means expect error.
	}

	testCases := []TestCase{
		{
			name: "format 1 events",
			txResp: abci.ResponseDeliverTx{
				GasUsed: 21000,
				Events: []abci.Event{
					{Type: "coin_received", Attributes: []abci.EventAttribute{
						{Key: "receiver", Value: "ethm12luku6uxehhak02py4rcz65zu0swh7wjun6msa"},
						{Key: "amount", Value: "1252860basetcro"},
					}},
					{Type: "coin_spent", Attributes: []abci.EventAttribute{
						{Key: "spender", Value: "ethm17xpfvakm2amg962yls6f84z3kell8c5lthdzgl"},
						{Key: "amount", Value: "1252860basetcro"},
					}},
					{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: "ethereumTxHash", Value: txHash.Hex()},
						{Key: "txIndex", Value: "10"},
						{Key: "amount", Value: "1000"},
						{Key: "txGasUsed", Value: "21000"},
						{Key: "txHash", Value: "14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"},
						{Key: "recipient", Value: "0x775b87ef5D82ca211811C1a02CE0fE0CA3a455d7"},
					}},
					{Type: "message", Attributes: []abci.EventAttribute{
						{Key: "action", Value: "/ehermint.evm.v1.MsgEthereumTx"},
						{Key: "key", Value: "ethm17xpfvakm2amg962yls6f84z3kell8c5lthdzgl"},
						{Key: "module", Value: "evm"},
						{Key: "sender", Value: address},
					}},
					{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: "ethereumTxHash", Value: txHash2.Hex()},
						{Key: "txIndex", Value: "11"},
						{Key: "amount", Value: "1000"},
						{Key: "txGasUsed", Value: "21000"},
						{Key: "txHash", Value: "14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"},
						{Key: "recipient", Value: "0x775b87ef5D82ca211811C1a02CE0fE0CA3a455d7"},
						{Key: "ethereumTxFailed", Value: "contract everted"},
					}},
					{Type: evm.EventTypeTxLog, Attributes: []abci.EventAttribute{}},
				},
			},
			wantEthTxs: []*ParsedTx{
				{
					MsgIndex:   0,
					Hash:       txHash,
					EthTxIndex: 10,
					GasUsed:    21000,
					Failed:     false,
				},
				{
					MsgIndex:   1,
					Hash:       txHash2,
					EthTxIndex: 11,
					GasUsed:    21000,
					Failed:     true,
				},
			},
		},
		{
			name: "format 2 events",
			txResp: abci.ResponseDeliverTx{
				GasUsed: 21000,
				Events: []abci.Event{
					{Type: "coin_received", Attributes: []abci.EventAttribute{
						{Key: "receiver", Value: "ethm12luku6uxehhak02py4rcz65zu0swh7wjun6msa"},
						{Key: "amount", Value: "1252860basetcro"},
					}},
					{Type: "coin_spent", Attributes: []abci.EventAttribute{
						{Key: "spender", Value: "ethm17xpfvakm2amg962yls6f84z3kell8c5lthdzgl"},
						{Key: "amount", Value: "1252860basetcro"},
					}},
					{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: "ethereumTxHash", Value: txHash.Hex()},
						{Key: "txIndex", Value: "0"},
					}},
					{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: "amount", Value: "1000"},
						{Key: "ethereumTxHash", Value: txHash.Hex()},
						{Key: "txIndex", Value: "0"},
						{Key: "txGasUsed", Value: "21000"},
						{Key: "txHash", Value: "14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"},
						{Key: "recipient", Value: "0x775b87ef5D82ca211811C1a02CE0fE0CA3a455d7"},
					}},
					{Type: "message", Attributes: []abci.EventAttribute{
						{Key: "action", Value: "/ehermint.evm.v1.MsgEthereumTx"},
						{Key: "key", Value: "ethm17xpfvakm2amg962yls6f84z3kell8c5lthdzgl"},
						{Key: "module", Value: "evm"},
						{Key: "sender", Value: address},
					}},
				},
			},
			wantEthTxs: []*ParsedTx{
				{
					MsgIndex:   0,
					Hash:       txHash,
					EthTxIndex: 0,
					GasUsed:    21000,
					Failed:     false,
				},
			},
		},
		{
			"format 1 events, failed",
			abci.ResponseDeliverTx{
				GasUsed: 21000,
				Events: []abci.Event{
					{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: "ethereumTxHash", Value: txHash.Hex()},
						{Key: "txIndex", Value: "10"},
						{Key: "amount", Value: "1000"},
						{Key: "txGasUsed", Value: "21000"},
						{Key: "txHash", Value: "14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"},
						{Key: "recipient", Value: "0x775b87ef5D82ca211811C1a02CE0fE0CA3a455d7"},
					}},
					{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: "ethereumTxHash", Value: txHash2.Hex()},
						{Key: "txIndex", Value: "10"},
						{Key: "amount", Value: "1000"},
						{Key: "txGasUsed", Value: "0x01"},
						{Key: "txHash", Value: "14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"},
						{Key: "recipient", Value: "0x775b87ef5D82ca211811C1a02CE0fE0CA3a455d7"},
						{Key: "ethereumTxFailed", Value: "contract everted"},
					}},
					{Type: evm.EventTypeTxLog, Attributes: []abci.EventAttribute{}},
				},
			},
			nil,
		},
		{
			name: "format 2 events failed",
			txResp: abci.ResponseDeliverTx{
				GasUsed: 21000,
				Events: []abci.Event{
					{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: "ethereumTxHash", Value: txHash.Hex()},
						{Key: "txIndex", Value: "10"},
					}},
					{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: "amount", Value: "1000"},
						{Key: "txGasUsed", Value: "0x01"},
						{Key: "txHash", Value: "14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"},
						{Key: "recipient", Value: "0x775b87ef5D82ca211811C1a02CE0fE0CA3a455d7"},
					}},
				},
			},
			wantEthTxs: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseTxResult(&tc.txResp, nil) //#nosec G601 -- fine for tests
			if tc.wantEthTxs == nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			for msgIndex, expTx := range tc.wantEthTxs {
				require.Equal(t, expTx, parsed.GetTxByMsgIndex(msgIndex))
				require.Equal(t, expTx, parsed.GetTxByHash(expTx.Hash))
				require.Equal(t, expTx, parsed.GetTxByTxIndex(int(expTx.EthTxIndex)))
			}
			// non-exists tx hash
			require.Nil(t, parsed.GetTxByHash(common.Hash{}))
			// out of range
			require.Nil(t, parsed.GetTxByMsgIndex(len(tc.wantEthTxs)))
			require.Nil(t, parsed.GetTxByTxIndex(99999999))
		})
	}
}

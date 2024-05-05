// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest

import (
	"math/big"
	"testing"

	cmt "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/eth/encoding"

	"github.com/cosmos/cosmos-sdk/client"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/evm"
)

// NewEthTxMsg: Helper that returns a valid instance of [*evm.MsgEthereumTx].
func NewEthTxMsg() *evm.MsgEthereumTx {
	return NewEthTxMsgs(1)[0]
}

func NewEthTxMsgs(count uint64) (ethTxMsgs []*evm.MsgEthereumTx) {
	ethAddr := testutil.NewEthAddr()
	startIdx := uint64(1)
	for nonce := startIdx; nonce-startIdx < count; nonce++ {
		ethTxMsgs = append(ethTxMsgs, evm.NewTx(&evm.EvmTxArgs{
			ChainID:  big.NewInt(1),
			Nonce:    nonce,
			To:       &ethAddr,
			GasLimit: 100000,
			GasPrice: big.NewInt(1),
			Input:    []byte("testinput"),
			Accesses: &gethcore.AccessList{},
		}),
		)
	}
	return ethTxMsgs
}

// NewEthTxMsgAsCmt: Helper that returns an Ethereum tx msg converted into
// tx bytes used in the Consensus Engine.
func NewEthTxMsgAsCmt(t *testing.T) (
	txBz cmt.Tx,
	ethTxMsgs []*evm.MsgEthereumTx,
	clientCtx client.Context,
) {
	// Build a TxBuilder that can understand Ethereum types
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	evm.RegisterInterfaces(encCfg.InterfaceRegistry)
	eth.RegisterInterfaces(encCfg.InterfaceRegistry)
	txConfig := encCfg.TxConfig
	clientCtx = client.Context{
		TxConfig:          txConfig,
		InterfaceRegistry: encCfg.InterfaceRegistry,
	}
	txBuilder := clientCtx.TxConfig.NewTxBuilder()

	// Build a consensus tx with a few Eth tx msgs
	ethTxMsgs = NewEthTxMsgs(3)
	assert.NoError(t,
		txBuilder.SetMsgs(ethTxMsgs[0], ethTxMsgs[1], ethTxMsgs[2]),
	)
	tx := txBuilder.GetTx()
	txBz, err := clientCtx.TxConfig.TxEncoder()(tx)
	assert.NoError(t, err)
	return txBz, ethTxMsgs, clientCtx
}

// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest

import (
	"math/big"
	"testing"

	cmt "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/x/evm"
)

// NewEthAccInfo returns an Ethereum private key, its corresponding Eth address,
// public key, and Nibiru address.
func NewEthAccInfo() EthPrivKeyAcc {
	privkey, _ := ethsecp256k1.GenerateKey()
	ethAddr := privkey.PubKey().Address()
	return EthPrivKeyAcc{
		EthAddr:       common.BytesToAddress(ethAddr.Bytes()),
		NibiruAddr:    sdk.AccAddress(ethAddr.Bytes()),
		PrivKey:       privkey,
		KeyringSigner: NewSigner(privkey),
	}
}

func EthAddrToNibiruAddr(ethAddr gethcommon.Address) sdk.AccAddress {
	return ethAddr.Bytes()
}

type EthPrivKeyAcc struct {
	EthAddr       common.Address
	NibiruAddr    sdk.AccAddress
	PrivKey       *ethsecp256k1.PrivKey
	KeyringSigner keyring.Signer
}

func (acc EthPrivKeyAcc) GethSigner(ethChainID *big.Int) gethcore.Signer {
	return gethcore.LatestSignerForChainID(ethChainID)
}

// NewEthTxMsg: Helper that returns a valid instance of [*evm.MsgEthereumTx].
func NewEthTxMsg() *evm.MsgEthereumTx {
	return NewEthTxMsgs(1)[0]
}

func NewEthTxMsgs(count uint64) (ethTxMsgs []*evm.MsgEthereumTx) {
	commonAddr := common.Address(NewEthAccInfo().EthAddr.Bytes())
	startIdx := uint64(1)
	for nonce := startIdx; nonce-startIdx < count; nonce++ {
		ethTxMsgs = append(ethTxMsgs, evm.NewTx(&evm.EvmTxArgs{
			ChainID:  big.NewInt(1),
			Nonce:    nonce,
			To:       &commonAddr,
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
	encCfg := app.MakeEncodingConfig()
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

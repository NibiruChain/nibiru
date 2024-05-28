// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest

import (
	"fmt"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/x/evm"
)

type GethTxType = uint8

var (
	GethTxType_LegacyTx     GethTxType = gethcore.LegacyTxType
	GethTxType_AccessListTx GethTxType = gethcore.AccessListTxType
	GethTxType_DynamicFeeTx GethTxType = gethcore.DynamicFeeTxType
)

func NewEthTx(
	deps *TestDeps, txData gethcore.TxData, nonce uint64,
) (ethCoreTx *gethcore.Transaction, err error) {
	ethCoreTx, err = NewEthTxUnsigned(deps, txData, nonce)
	if err != nil {
		return ethCoreTx, err
	}

	sig, _, err := deps.Sender.KeyringSigner.SignByAddress(
		deps.Sender.NibiruAddr, ethCoreTx.Hash().Bytes(),
	)
	if err != nil {
		return ethCoreTx, err
	}

	return ethCoreTx.WithSignature(deps.GethSigner(), sig)
}

func NewEthTxUnsigned(
	deps *TestDeps, txData gethcore.TxData, nonce uint64,
) (ethCoreTx *gethcore.Transaction, err error) {
	switch typedTxData := txData.(type) {
	case *gethcore.LegacyTx:
		typedTxData.Nonce = nonce
		ethCoreTx = gethcore.NewTx(typedTxData)
	case *gethcore.AccessListTx:
		typedTxData.Nonce = nonce
		ethCoreTx = gethcore.NewTx(typedTxData)
	case *gethcore.DynamicFeeTx:
		typedTxData.Nonce = nonce
		ethCoreTx = gethcore.NewTx(typedTxData)
	default:
		return ethCoreTx, fmt.Errorf("received unknown tx type in NewEthTxUnsigned")
	}
	return ethCoreTx, err
}

func TxTemplateAccessListTx() *gethcore.AccessListTx {
	return &gethcore.AccessListTx{
		GasPrice: big.NewInt(1),
		Gas:      gethparams.TxGas,
		To:       &gethcommon.Address{},
		Value:    big.NewInt(0),
		Data:     []byte{},
	}
}

func TxTemplateLegacyTx() *gethcore.LegacyTx {
	return &gethcore.LegacyTx{
		GasPrice: big.NewInt(1),
		Gas:      gethparams.TxGas,
		To:       &gethcommon.Address{},
		Value:    big.NewInt(0),
		Data:     []byte{},
	}
}

func TxTemplateDynamicFeeTx() *gethcore.DynamicFeeTx {
	return &gethcore.DynamicFeeTx{
		GasFeeCap: big.NewInt(10),
		GasTipCap: big.NewInt(2),
		Gas:       gethparams.TxGas,
		To:        &gethcommon.Address{},
		Value:     big.NewInt(0),
		Data:      []byte{},
	}
}

func NewEthTxMsgFromTxData(
	deps *TestDeps,
	txType GethTxType,
	innerTxData []byte,
	nonce uint64,
	accessList gethcore.AccessList,
) (*evm.MsgEthereumTx, error) {
	if innerTxData == nil {
		innerTxData = []byte{}
	}

	var ethCoreTx *gethcore.Transaction
	switch txType {
	case gethcore.LegacyTxType:
		innerTx := TxTemplateLegacyTx()
		innerTx.Nonce = nonce
		innerTx.Data = innerTxData
		ethCoreTx = gethcore.NewTx(innerTx)
	case gethcore.AccessListTxType:
		innerTx := TxTemplateAccessListTx()
		innerTx.Nonce = nonce
		innerTx.Data = innerTxData
		innerTx.AccessList = accessList
		ethCoreTx = gethcore.NewTx(innerTx)
	case gethcore.DynamicFeeTxType:
		innerTx := TxTemplateDynamicFeeTx()
		innerTx.Nonce = nonce
		innerTx.Data = innerTxData
		innerTx.AccessList = accessList
		ethCoreTx = gethcore.NewTx(innerTx)
	default:
		return nil, fmt.Errorf(
			"received unknown tx type (%v) in NewEthTxMsgFromTxData", txType)
	}

	ethTxMsg := new(evm.MsgEthereumTx)
	if err := ethTxMsg.FromEthereumTx(ethCoreTx); err != nil {
		return ethTxMsg, err
	}

	ethTxMsg.From = deps.Sender.EthAddr.Hex()
	return ethTxMsg, ethTxMsg.Sign(deps.GethSigner(), deps.Sender.KeyringSigner)
}

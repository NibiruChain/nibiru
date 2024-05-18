// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest

import (
	"fmt"

	gethcore "github.com/ethereum/go-ethereum/core/types"
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
		return ethCoreTx, fmt.Errorf("received unknown tx type in NewCoreTx")
	}
	return ethCoreTx, err
}

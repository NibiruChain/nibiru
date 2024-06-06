// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	srvconfig "github.com/NibiruChain/nibiru/app/server/config"

	"github.com/NibiruChain/nibiru/x/evm"
)

type GethTxType = uint8

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

// ExecuteNibiTransfer executes nibi transfer
func ExecuteNibiTransfer(deps *TestDeps, t *testing.T) *evm.MsgEthereumTx {
	nonce := deps.StateDB().GetNonce(deps.Sender.EthAddr)
	recipient := NewEthAccInfo().EthAddr

	txArgs := evm.JsonTxArgs{
		From:  &deps.Sender.EthAddr,
		To:    &recipient,
		Nonce: (*hexutil.Uint64)(&nonce),
	}
	ethTxMsg, err := GenerateAndSignEthTxMsg(txArgs, deps)
	require.NoError(t, err)

	resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
	require.NoError(t, err)
	require.Empty(t, resp.VmError)
	return ethTxMsg
}

// ExecuteERC20Transfer deploys contract, executes transfer and returns tx hash
func ExecuteERC20Transfer(deps *TestDeps, t *testing.T) (*evm.MsgEthereumTx, []*evm.MsgEthereumTx) {
	// TX 1: Deploy ERC-20 contract
	contractData := SmartContract_FunToken.Load(t)
	nonce := deps.StateDB().GetNonce(deps.Sender.EthAddr)
	txArgs := evm.JsonTxArgs{
		From:  &deps.Sender.EthAddr,
		Nonce: (*hexutil.Uint64)(&nonce),
		Data:  (*hexutil.Bytes)(&contractData.Bytecode),
	}
	ethTxMsg, err := GenerateAndSignEthTxMsg(txArgs, deps)
	require.NoError(t, err)

	resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
	require.NoError(t, err)
	require.Empty(t, resp.VmError)

	// Contract address is deterministic
	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	deps.Chain.Commit()
	predecessors := []*evm.MsgEthereumTx{
		ethTxMsg,
	}

	// TX 2: execute ERC-20 contract transfer
	input, err := contractData.ABI.Pack(
		"transfer", NewEthAccInfo().EthAddr, new(big.Int).SetUint64(1000),
	)
	require.NoError(t, err)
	nonce = deps.StateDB().GetNonce(deps.Sender.EthAddr)
	txArgs = evm.JsonTxArgs{
		From:  &deps.Sender.EthAddr,
		To:    &contractAddress,
		Nonce: (*hexutil.Uint64)(&nonce),
		Data:  (*hexutil.Bytes)(&input),
	}
	ethTxMsg, err = GenerateAndSignEthTxMsg(txArgs, deps)
	require.NoError(t, err)

	resp, err = deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
	require.NoError(t, err)
	require.Empty(t, resp.VmError)

	return ethTxMsg, predecessors
}

// GenerateAndSignEthTxMsg estimates gas, sets gas limit and sings the tx
func GenerateAndSignEthTxMsg(txArgs evm.JsonTxArgs, deps *TestDeps) (*evm.MsgEthereumTx, error) {
	estimateArgs, err := json.Marshal(&txArgs)
	if err != nil {
		return nil, err
	}
	res, err := deps.Chain.EvmKeeper.EstimateGas(deps.GoCtx(), &evm.EthCallRequest{
		Args:            estimateArgs,
		GasCap:          srvconfig.DefaultGasCap,
		ProposerAddress: []byte{},
		ChainId:         deps.Chain.EvmKeeper.EthChainID(deps.Ctx).Int64(),
	})
	if err != nil {
		return nil, err
	}
	txArgs.Gas = (*hexutil.Uint64)(&res.Gas)

	txMsg := txArgs.ToTransaction()
	gethSigner := deps.Sender.GethSigner(deps.Chain.EvmKeeper.EthChainID(deps.Ctx))
	keyringSigner := deps.Sender.KeyringSigner
	return txMsg, txMsg.Sign(gethSigner, keyringSigner)
}

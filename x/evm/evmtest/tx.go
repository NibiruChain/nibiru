// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	srvconfig "github.com/NibiruChain/nibiru/app/server/config"

	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"
)

type GethTxType = uint8

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
	to *gethcommon.Address,
	value *big.Int,
	gas uint64,
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
		innerTx.To = to
		innerTx.Value = value
		innerTx.Gas = gas
		ethCoreTx = gethcore.NewTx(innerTx)
	case gethcore.AccessListTxType:
		innerTx := TxTemplateAccessListTx()
		innerTx.Nonce = nonce
		innerTx.Data = innerTxData
		innerTx.AccessList = accessList
		innerTx.To = to
		innerTx.Value = value
		innerTx.Gas = gas
		ethCoreTx = gethcore.NewTx(innerTx)
	case gethcore.DynamicFeeTxType:
		innerTx := TxTemplateDynamicFeeTx()
		innerTx.Nonce = nonce
		innerTx.Data = innerTxData
		innerTx.To = to
		innerTx.Value = value
		innerTx.Gas = gas
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

type DeployContractResult struct {
	TxResp       *evm.MsgEthereumTxResponse
	EthTxMsg     *evm.MsgEthereumTx
	ContractData embeds.CompiledEvmContract
	Nonce        uint64
	ContractAddr gethcommon.Address
}

func DeployContract(
	deps *TestDeps,
	contract embeds.CompiledEvmContract,
	t *testing.T,
	args ...any,
) (result DeployContractResult, err error) {
	// Use contract args
	packedArgs, err := contract.ABI.Pack("", args...)
	if err != nil {
		err = errors.Wrap(err, "failed to pack ABI args")
		return
	}
	bytecodeForCall := append(contract.Bytecode, packedArgs...)

	nonce := deps.StateDB().GetNonce(deps.Sender.EthAddr)
	jsonTxArgs := evm.JsonTxArgs{
		Nonce: (*hexutil.Uint64)(&nonce),
		Input: (*hexutil.Bytes)(&bytecodeForCall),
		From:  &deps.Sender.EthAddr,
	}
	ethTxMsg, err := GenerateAndSignEthTxMsg(jsonTxArgs, deps)
	require.NoError(t, err)

	resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
	require.NoError(t, err)
	require.Empty(t, resp.VmError)

	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)

	return DeployContractResult{
		TxResp:       resp,
		EthTxMsg:     ethTxMsg,
		ContractData: contract,
		Nonce:        nonce,
		ContractAddr: contractAddress,
	}, nil
}

// DeployAndExecuteERC20Transfer deploys contract, executes transfer and returns tx hash
func DeployAndExecuteERC20Transfer(
	deps *TestDeps, t *testing.T,
) (*evm.MsgEthereumTx, []*evm.MsgEthereumTx) {
	// TX 1: Deploy ERC-20 contract
	deployResp, err := DeployContract(deps, embeds.SmartContract_TestERC20, t)
	require.NoError(t, err)
	contractData := deployResp.ContractData
	nonce := deployResp.Nonce

	// Contract address is deterministic
	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	deps.Chain.Commit()
	predecessors := []*evm.MsgEthereumTx{
		deployResp.EthTxMsg,
	}

	// TX 2: execute ERC-20 contract transfer
	input, err := contractData.ABI.Pack(
		"transfer", NewEthAccInfo().EthAddr, new(big.Int).SetUint64(1000),
	)
	require.NoError(t, err)
	nonce = deps.StateDB().GetNonce(deps.Sender.EthAddr)
	txArgs := evm.JsonTxArgs{
		From:  &deps.Sender.EthAddr,
		To:    &contractAddress,
		Nonce: (*hexutil.Uint64)(&nonce),
		Data:  (*hexutil.Bytes)(&input),
	}
	ethTxMsg, err := GenerateAndSignEthTxMsg(txArgs, deps)
	require.NoError(t, err)

	resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
	require.NoError(t, err)
	require.Empty(t, resp.VmError)

	return ethTxMsg, predecessors
}

// GenerateAndSignEthTxMsg estimates gas, sets gas limit and sings the tx
func GenerateAndSignEthTxMsg(
	txArgs evm.JsonTxArgs, deps *TestDeps,
) (*evm.MsgEthereumTx, error) {
	estimateArgs, err := json.Marshal(&txArgs)
	if err != nil {
		return nil, err
	}
	res, err := deps.Chain.EvmKeeper.EstimateGas(deps.GoCtx(), &evm.EthCallRequest{
		Args:            estimateArgs,
		GasCap:          srvconfig.DefaultEthCallGasLimit,
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

func TransferWei(
	deps *TestDeps,
	to gethcommon.Address,
	amountWei *big.Int,
) error {
	ethAcc := deps.Sender
	var innerTxData []byte = nil
	var accessList gethcore.AccessList = nil
	ethTxMsg, err := NewEthTxMsgFromTxData(
		deps,
		gethcore.LegacyTxType,
		innerTxData,
		deps.StateDB().GetNonce(ethAcc.EthAddr),
		&to,
		amountWei,
		gethparams.TxGas,
		accessList,
	)
	if err != nil {
		return fmt.Errorf("error while transferring wei: %w", err)
	}

	_, err = deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
	if err != nil {
		return fmt.Errorf("error while transferring wei: %w", err)
	}
	return err
}

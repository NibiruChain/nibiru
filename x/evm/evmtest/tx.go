// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	srvconfig "github.com/NibiruChain/nibiru/v2/app/server/config"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
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
	recipient := NewEthPrivAcc().EthAddr

	txArgs := evm.JsonTxArgs{
		From:  &deps.Sender.EthAddr,
		To:    &recipient,
		Nonce: (*hexutil.Uint64)(&nonce),
	}
	ethTxMsg, gethSigner, krSigner, err := GenerateEthTxMsgAndSigner(txArgs, deps, deps.Sender)
	require.NoError(t, err)
	err = ethTxMsg.Sign(gethSigner, krSigner)
	require.NoError(t, err)

	resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(deps.Ctx), ethTxMsg)
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
	args ...any,
) (result *DeployContractResult, err error) {
	// Use contract args
	packedArgs, err := contract.ABI.Pack("", args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pack contract args")
	}
	bytecodeForCall := append(contract.Bytecode, packedArgs...)

	nonce := deps.StateDB().GetNonce(deps.Sender.EthAddr)
	ethTxMsg, gethSigner, krSigner, err := GenerateEthTxMsgAndSigner(
		evm.JsonTxArgs{
			Nonce: (*hexutil.Uint64)(&nonce),
			Input: (*hexutil.Bytes)(&bytecodeForCall),
			From:  &deps.Sender.EthAddr,
		}, deps, deps.Sender,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate and sign eth tx msg")
	} else if err := ethTxMsg.Sign(gethSigner, krSigner); err != nil {
		return nil, errors.Wrap(err, "failed to generate and sign eth tx msg")
	}

	resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(deps.Ctx), ethTxMsg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute ethereum tx")
	}
	if resp.VmError != "" {
		return nil, fmt.Errorf("vm error: %s", resp.VmError)
	}

	return &DeployContractResult{
		TxResp:       resp,
		EthTxMsg:     ethTxMsg,
		ContractData: contract,
		Nonce:        nonce,
		ContractAddr: crypto.CreateAddress(deps.Sender.EthAddr, nonce),
	}, nil
}

// DeployAndExecuteERC20Transfer deploys contract, executes transfer and returns tx hash
func DeployAndExecuteERC20Transfer(
	deps *TestDeps, t *testing.T,
) (erc20Transfer *evm.MsgEthereumTx, predecessors []*evm.MsgEthereumTx) {
	// TX 1: Deploy ERC-20 contract
	deployResp, err := DeployContract(deps, embeds.SmartContract_TestERC20)
	require.NoError(t, err)
	contractData := deployResp.ContractData
	nonce := deployResp.Nonce

	// Contract address is deterministic
	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	deps.App.Commit()
	predecessors = []*evm.MsgEthereumTx{
		deployResp.EthTxMsg,
	}

	// TX 2: execute ERC-20 contract transfer
	input, err := contractData.ABI.Pack(
		"transfer", NewEthPrivAcc().EthAddr, new(big.Int).SetUint64(1000),
	)
	require.NoError(t, err)
	nonce = deps.StateDB().GetNonce(deps.Sender.EthAddr)
	txArgs := evm.JsonTxArgs{
		From:  &deps.Sender.EthAddr,
		To:    &contractAddress,
		Nonce: (*hexutil.Uint64)(&nonce),
		Data:  (*hexutil.Bytes)(&input),
	}
	erc20Transfer, gethSigner, krSigner, err := GenerateEthTxMsgAndSigner(txArgs, deps, deps.Sender)
	require.NoError(t, err)
	err = erc20Transfer.Sign(gethSigner, krSigner)
	require.NoError(t, err)

	resp, err := deps.App.EvmKeeper.EthereumTx(deps.GoCtx(), erc20Transfer)
	require.NoError(t, err)
	require.Empty(t, resp.VmError)

	return erc20Transfer, predecessors
}

func CallContractTx(
	deps *TestDeps,
	contractAddr gethcommon.Address,
	input []byte,
	sender EthPrivKeyAcc,
) (ethTxMsg *evm.MsgEthereumTx, resp *evm.MsgEthereumTxResponse, err error) {
	nonce := deps.StateDB().GetNonce(sender.EthAddr)
	ethTxMsg, gethSigner, krSigner, err := GenerateEthTxMsgAndSigner(evm.JsonTxArgs{
		From:  &sender.EthAddr,
		To:    &contractAddr,
		Nonce: (*hexutil.Uint64)(&nonce),
		Data:  (*hexutil.Bytes)(&input),
	}, deps, sender)
	if err != nil {
		err = fmt.Errorf("CallContract error during tx generation: %w", err)
		return
	}

	txConfig := deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.HexToHash(ethTxMsg.Hash))
	stateDB := statedb.New(deps.Ctx, &deps.EvmKeeper, txConfig)
	err = stateDB.Commit()
	if err != nil {
		return
	}

	err = ethTxMsg.Sign(gethSigner, krSigner)
	if err != nil {
		err = fmt.Errorf("CallContract error during signature: %w", err)
		return
	}

	resp, err = deps.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
	return ethTxMsg, resp, err
}

// GenerateEthTxMsgAndSigner estimates gas, sets gas limit and returns signer for
// the tx.
//
// Usage:
//
//	```go
//	evmTxMsg, gethSigner, krSigner, _ := GenerateEthTxMsgAndSigner(
//	    jsonTxArgs, &deps, sender,
//	)
//	err := evmTxMsg.Sign(gethSigner, sender.KeyringSigner)
//	```
func GenerateEthTxMsgAndSigner(
	jsonTxArgs evm.JsonTxArgs, deps *TestDeps, sender EthPrivKeyAcc,
) (evmTxMsg *evm.MsgEthereumTx, gethSigner gethcore.Signer, krSigner keyring.Signer, err error) {
	estimateArgs, err := json.Marshal(&jsonTxArgs)
	if err != nil {
		return
	}
	res, err := deps.App.EvmKeeper.EstimateGas(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.EthCallRequest{
			Args:            estimateArgs,
			GasCap:          srvconfig.DefaultEthCallGasLimit,
			ProposerAddress: []byte{},
			ChainId:         deps.App.EvmKeeper.EthChainID(deps.Ctx).Int64(),
		},
	)
	if err != nil {
		return
	}
	jsonTxArgs.Gas = (*hexutil.Uint64)(&res.Gas)

	evmTxMsg = jsonTxArgs.ToMsgEthTx()
	gethSigner = gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx))
	return evmTxMsg, gethSigner, sender.KeyringSigner, nil
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

	_, err = deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(deps.Ctx), ethTxMsg)
	if err != nil {
		return fmt.Errorf("error while transferring wei: %w", err)
	}
	return err
}

// ValidLegacyTx: Useful initial condition for tests
// Exported only for use in tests.
func ValidLegacyTx() *evm.LegacyTx {
	sdkInt := sdkmath.NewIntFromBigInt(evm.NativeToWei(big.NewInt(420)))
	return &evm.LegacyTx{
		Nonce:    24,
		GasLimit: 50_000,
		To:       gethcommon.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed").Hex(),
		GasPrice: &sdkInt,
		Amount:   &sdkInt,
		Data:     []byte{},
		V:        []byte{},
		R:        []byte{},
		S:        []byte{},
	}
}

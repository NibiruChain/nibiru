// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

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
)

// ExecuteNibiTransfer executes nibi transfer
func ExecuteNibiTransfer(deps *TestDeps, t *testing.T) *evm.MsgEthereumTx {
	nonce := deps.NewStateDB().GetNonce(deps.Sender.EthAddr)
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

	nonce := deps.NewStateDB().GetNonce(deps.Sender.EthAddr)
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
) (
	erc20Transfer *evm.MsgEthereumTx,
	predecessors []*evm.MsgEthereumTx,
	contractAddr gethcommon.Address,
) {
	// TX 1: Deploy ERC-20 contract
	deployResp, err := DeployContract(deps, embeds.SmartContract_TestERC20)
	require.NoError(t, err)
	contractData := deployResp.ContractData
	nonce := deployResp.Nonce

	// Contract address is deterministic
	contractAddr = crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	deps.App.Commit()
	predecessors = []*evm.MsgEthereumTx{
		deployResp.EthTxMsg,
	}

	// TX 2: execute ERC-20 contract transfer
	input, err := contractData.ABI.Pack(
		"transfer", NewEthPrivAcc().EthAddr, new(big.Int).SetUint64(1000),
	)
	require.NoError(t, err)
	nonce = deps.NewStateDB().GetNonce(deps.Sender.EthAddr)
	txArgs := evm.JsonTxArgs{
		From:  &deps.Sender.EthAddr,
		To:    &contractAddr,
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

	return erc20Transfer, predecessors, contractAddr
}

func CallContractTx(
	deps *TestDeps,
	contractAddr gethcommon.Address,
	input []byte,
	sender EthPrivKeyAcc,
) (ethTxMsg *evm.MsgEthereumTx, resp *evm.MsgEthereumTxResponse, err error) {
	nonce := deps.NewStateDB().GetNonce(sender.EthAddr)
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

	err = ethTxMsg.Sign(gethSigner, krSigner)
	if err != nil {
		err = fmt.Errorf("CallContract error during signature: %w", err)
		return
	}

	resp, err = deps.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
	return ethTxMsg, resp, err
}

var DefaultEthCallGasLimit = srvconfig.DefaultEthCallGasLimit

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
		deps.NewStateDB().GetNonce(ethAcc.EthAddr),
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

// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"encoding/json"
	"math/big"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	serverconfig "github.com/NibiruChain/nibiru/app/server/config"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

func (k Keeper) FindERC20Metadata(
	ctx sdk.Context,
	contract gethcommon.Address,
) (info ERC20Metadata, err error) {
	var abi gethabi.ABI = embeds.EmbeddedContractERC20Minter.ABI

	errs := []error{}

	// Load name, symbol, decimals
	name, err := k.LoadERC20Name(ctx, abi, contract)
	errs = append(errs, err)
	symbol, err := k.LoadERC20Symbol(ctx, abi, contract)
	errs = append(errs, err)
	decimals, err := k.LoadERC20Decimals(ctx, abi, contract)
	errs = append(errs, err)

	err = common.CombineErrors(errs...)
	if err != nil {
		return info, errors.Wrap(err, "failed to \"FindERC20Metadata\"")
	}

	return ERC20Metadata{
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}, nil
}

type ERC20Metadata struct {
	Name     string
	Symbol   string
	Decimals uint8
}

type (
	ERC20String struct{ Value string }
	ERC20Uint8  struct{ Value uint8 }
	ERC20Bool   struct{ Value bool }
)

// CallContract invokes a smart contract on the method specified by [methodName]
// using the given [args].
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - abi: The ABI (Application Binary Interface) of the smart contract.
//   - fromAcc: The Ethereum address of the account initiating the contract call.
//   - contract: Pointer to the Ethereum address of the contract to be called.
//   - commit: Boolean flag indicating whether to commit the transaction (true) or simulate it (false).
//   - methodName: The name of the contract method to be called.
//   - args: Variadic parameter for the arguments to be passed to the contract method.
//
// Note: This function handles both contract method calls and simulations,
// depending on the 'commit' parameter. It uses a default gas limit for
// simulations and estimates gas for actual transactions.
func (k Keeper) CallContract(
	ctx sdk.Context,
	abi gethabi.ABI,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	commit bool,
	methodName string,
	args ...any,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	contractData, err := abi.Pack(methodName, args...)
	if err != nil {
		return evmResp, err
	}

	nonce := k.GetAccNonce(ctx, fromAcc)

	// FIXME: Is this gas limit convention useful?
	gasLimit := serverconfig.DefaultEthCallGasLimit
	if commit {
		jsonArgs, err := json.Marshal(evm.JsonTxArgs{
			From: &fromAcc,
			To:   contract,
			Data: (*hexutil.Bytes)(&contractData),
		})
		if err != nil {
			return evmResp, err // TODO: UD-DEBUG: ...
		}

		gasRes, err := k.EstimateGasForEvmCallType(
			sdk.WrapSDKContext(ctx),
			&evm.EthCallRequest{
				Args:   jsonArgs,
				GasCap: gasLimit,
			},
			evm.CallTypeSmart,
		)
		if err != nil {
			return evmResp, err
		}

		gasLimit = gasRes.Gas
	}

	unusedBitInt := big.NewInt(0)
	evmMsg := gethcore.NewMessage(
		fromAcc,
		contract,
		nonce,
		unusedBitInt, // amount
		gasLimit,
		unusedBitInt, // gasFeeCap
		unusedBitInt, // gasTipCap
		unusedBitInt, // gasPrice
		contractData,
		gethcore.AccessList{},
		!commit, // isFake
	)

	// Apply EVM message
	cfg, err := k.GetEVMConfig(
		ctx,
		sdk.ConsAddress(ctx.BlockHeader().ProposerAddress),
		k.EthChainID(ctx),
	)
	if err != nil {
		return evmResp, errors.Wrap(err, "failed to load evm config")
	}
	txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash()))
	evmResp, err = k.ApplyEvmMsg(
		ctx, evmMsg, evm.NewNoOpTracer(), commit, cfg, txConfig,
	)
	if err != nil {
		return evmResp, err
	}

	if evmResp.Failed() {
		return evmResp, errors.Wrap(err, evmResp.VmError)
	}

	return evmResp, err
}

func (k Keeper) LoadERC20Name(
	ctx sdk.Context, abi gethabi.ABI, erc20 gethcommon.Address,
) (out string, err error) {
	return k.loadERC20String(ctx, abi, erc20, "name")
}

func (k Keeper) LoadERC20Symbol(
	ctx sdk.Context, abi gethabi.ABI, erc20 gethcommon.Address,
) (out string, err error) {
	return k.loadERC20String(ctx, abi, erc20, "symbol")
}

func (k Keeper) LoadERC20Decimals(
	ctx sdk.Context, abi gethabi.ABI, erc20 gethcommon.Address,
) (out uint8, err error) {
	return k.loadERC20Uint8(ctx, abi, erc20, "decimals")
}

func (k Keeper) loadERC20String(
	ctx sdk.Context,
	erc20Abi gethabi.ABI,
	erc20Contract gethcommon.Address,
	methodName string,
) (out string, err error) {
	res, err := k.CallContract(
		ctx, erc20Abi,
		evm.ModuleAddressEVM(),
		&erc20Contract,
		false, methodName,
	)
	if err != nil {
		return out, err
	}

	erc20string := new(ERC20String)
	err = erc20Abi.UnpackIntoInterface(
		erc20string, methodName, res.Ret,
	)
	if err != nil {
		return out, err
	}
	return erc20string.Value, err
}

func (k Keeper) loadERC20Uint8(
	ctx sdk.Context,
	erc20Abi gethabi.ABI,
	erc20Contract gethcommon.Address,
	methodName string,
) (out uint8, err error) {
	res, err := k.CallContract(
		ctx, erc20Abi,
		evm.ModuleAddressEVM(),
		&erc20Contract,
		false, methodName,
	)
	if err != nil {
		return out, err
	}

	erc20uint8 := new(ERC20Uint8)
	err = erc20Abi.UnpackIntoInterface(
		erc20uint8, methodName, res.Ret,
	)
	if err != nil {
		return out, err
	}
	return erc20uint8.Value, err
}

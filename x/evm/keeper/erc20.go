// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	serverconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

// ERC20 returns a mutable reference to the keeper with an ERC20 contract ABI and
// Go functions corresponding to contract calls in the ERC20 standard like "mint"
// and "transfer" in the ERC20 standard.
func (k Keeper) ERC20() erc20Calls {
	return erc20Calls{
		Keeper: &k,
		ABI:    embeds.SmartContract_ERC20Minter.ABI,
	}
}

type erc20Calls struct {
	*Keeper
	ABI *gethabi.ABI
}

/*
Mint implements "ERC20Minter.mint" from ERC20Minter.sol.
See [nibiru/x/evm/embeds].

	```solidity
	/// @dev Moves `amount` tokens from the caller's account to `to`.
	/// Returns a boolean value indicating whether the operation succeeded.
	/// Emits a {Transfer} event.
	function mint(address to, uint256 amount) public virtual onlyOwner {
	  _mint(to, amount);
	}
	```

[nibiru/x/evm/embeds]: https://github.com/NibiruChain/nibiru/v2/tree/main/x/evm/embeds
*/
func (e erc20Calls) Mint(
	contract, from, to gethcommon.Address, amount *big.Int,
	ctx sdk.Context,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	input, err := e.ABI.Pack("mint", to, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to pack ABI args: %w", err)
	}
	return e.CallContractWithInput(ctx, from, &contract, true, input)
}

/*
Transfer implements "ERC20.transfer"

	```solidity
	/// @dev Moves `amount` tokens from the caller's account to `to`.
	/// Returns a boolean value indicating whether the operation succeeded.
	/// Emits a {Transfer} event.
	function transfer(address to, uint256 amount) external returns (bool);
	```
*/
func (e erc20Calls) Transfer(
	contract, from, to gethcommon.Address, amount *big.Int,
	ctx sdk.Context,
) (success bool, err error, received *big.Int) {
	received = big.NewInt(0)

	recipientBalanceBefore, err := e.BalanceOf(contract, to, ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to retrieve recipient balance"), received
	}

	input, err := e.ABI.Pack("transfer", to, amount)
	if err != nil {
		return false, fmt.Errorf("failed to pack ABI args: %w", err), received
	}
	resp, err := e.CallContractWithInput(ctx, from, &contract, true, input)
	if err != nil {
		return false, err, received
	}

	recipientBalanceAfter, err := e.BalanceOf(contract, to, ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to retrieve recipient balance"), received
	}

	received = new(big.Int).Sub(recipientBalanceAfter, recipientBalanceBefore)

	// we can't check that received = amount because the recipient could have
	// a transfer fee or other deductions. We can only check that the recipient
	// received some tokens
	if received.Sign() <= 0 {
		return false, fmt.Errorf("no (or negative) ERC20 tokens were received by the recipient"), received
	}

	var erc20Bool ERC20Bool
	err = e.ABI.UnpackIntoInterface(&erc20Bool, "transfer", resp.Ret)

	// per erc20 standard, the transfer function should return a boolean value
	// indicating whether the operation succeeded. If the unpacking failed, we
	// need to check the recipient balance to determine if the transfer was successful.
	if err == nil {
		// should be true if the transfer was successful but we do it anyway
		// to respect the contract's return value
		success = erc20Bool.Value

		return success, nil, received
	}

	success = true
	return
}

// BalanceOf retrieves the balance of an ERC20 token for a specific account.
// Implements "ERC20.balanceOf".
func (e erc20Calls) BalanceOf(
	contract, account gethcommon.Address,
	ctx sdk.Context,
) (out *big.Int, err error) {
	return e.LoadERC20BigInt(ctx, e.ABI, contract, "balanceOf", account)
}

/*
Burn implements "ERC20Burnable.burn"

	```solidity
	/// @dev Destroys `amount` tokens from the caller.
	function burn(uint256 amount) public virtual {
	```
*/
func (e erc20Calls) Burn(
	contract, from gethcommon.Address, amount *big.Int,
	ctx sdk.Context,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	input, err := e.ABI.Pack("burn", amount)
	if err != nil {
		return
	}
	commit := true
	return e.CallContractWithInput(ctx, from, &contract, commit, input)
}

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
	abi *gethabi.ABI,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	commit bool,
	methodName string,
	args ...any,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	contractInput, err := abi.Pack(methodName, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack ABI args: %w", err)
	}
	return k.CallContractWithInput(ctx, fromAcc, contract, commit, contractInput)
}

// CallContractWithInput invokes a smart contract with the given [contractInput].
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - fromAcc: The Ethereum address of the account initiating the contract call.
//   - contract: Pointer to the Ethereum address of the contract to be called.
//   - commit: Boolean flag indicating whether to commit the transaction (true) or simulate it (false).
//   - contractInput: Hexadecimal-encoded bytes use as input data to the call.
//
// Note: This function handles both contract method calls and simulations,
// depending on the 'commit' parameter. It uses a default gas limit for
// simulations and estimates gas for actual transactions.
func (k Keeper) CallContractWithInput(
	ctx sdk.Context,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	commit bool,
	contractInput []byte,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	// This is a `defer` pattern to add behavior that runs in the case that the error is
	// non-nil, creating a concise way to add extra information.
	defer func() {
		if err != nil {
			err = fmt.Errorf("CallContractError: %w", err)
		}
	}()
	nonce := k.GetAccNonce(ctx, fromAcc)

	gasLimit := serverconfig.DefaultEthCallGasLimit
	gasLimit, err = computeCommitGasLimit(
		commit, gasLimit, &fromAcc, contract, contractInput, k, ctx,
	)
	if err != nil {
		return nil, err
	}

	unusedBigInt := big.NewInt(0)
	evmMsg := gethcore.NewMessage(
		fromAcc,
		contract,
		nonce,
		unusedBigInt, // amount
		gasLimit,
		unusedBigInt, // gasFeeCap
		unusedBigInt, // gasTipCap
		unusedBigInt, // gasPrice
		contractInput,
		gethcore.AccessList{},
		!commit, // isFake
	)

	// Apply EVM message
	evmCfg, err := k.GetEVMConfig(
		ctx,
		sdk.ConsAddress(ctx.BlockHeader().ProposerAddress),
		k.EthChainID(ctx),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load evm config")
	}

	txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash()))
	evmResp, err = k.ApplyEvmMsg(
		ctx, evmMsg, evm.NewNoOpTracer(), commit, evmCfg, txConfig,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to apply EVM message")
	}

	if evmResp.Failed() {
		return nil, errors.Wrapf(err, "EVM execution failed: %s", evmResp.VmError)
	}

	return evmResp, err
}

// computeCommitGasLimit: If the transition is meant to mutate state, this
// function computes an appopriates gas limit for the call with "contractInput"
// bytes against the given contract address.
//
// ℹ️ This creates a cached context for gas estimation, which is essential
// because state transitions can occur outside of the EVM that are triggered
// by Ethereum transactions, like in the case of precompiled contract or
// custom EVM hook that runs after tx execution. Gas estimation in that case
// could mutate the "ctx" object and result in falty resulting state, so we
// must cache here to avoid this issue.
func computeCommitGasLimit(
	commit bool,
	gasLimit uint64,
	fromAcc, contract *gethcommon.Address,
	contractInput []byte,
	k Keeper,
	ctx sdk.Context,
) (newGasLimit uint64, err error) {
	if !commit {
		return gasLimit, nil
	}

	cachedCtx, _ := ctx.CacheContext() // IMPORTANT!

	jsonArgs, err := json.Marshal(evm.JsonTxArgs{
		From: fromAcc,
		To:   contract,
		Data: (*hexutil.Bytes)(&contractInput),
	})
	if err != nil {
		return gasLimit, fmt.Errorf("failed compute gas limit to marshal tx args: %w", err)
	}

	gasRes, err := k.EstimateGasForEvmCallType(
		sdk.WrapSDKContext(cachedCtx),
		&evm.EthCallRequest{
			Args:   jsonArgs,
			GasCap: gasLimit,
		},
		evm.CallTypeSmart,
	)
	if err != nil {
		return gasLimit, fmt.Errorf("failed to compute gas limit: %w", err)
	}

	return gasRes.Gas, nil
}

func (k Keeper) LoadERC20Name(
	ctx sdk.Context, abi *gethabi.ABI, erc20 gethcommon.Address,
) (out string, err error) {
	return k.LoadERC20String(ctx, abi, erc20, "name")
}

func (k Keeper) LoadERC20Symbol(
	ctx sdk.Context, abi *gethabi.ABI, erc20 gethcommon.Address,
) (out string, err error) {
	return k.LoadERC20String(ctx, abi, erc20, "symbol")
}

func (k Keeper) LoadERC20Decimals(
	ctx sdk.Context, abi *gethabi.ABI, erc20 gethcommon.Address,
) (out uint8, err error) {
	return k.loadERC20Uint8(ctx, abi, erc20, "decimals")
}

func (k Keeper) LoadERC20String(
	ctx sdk.Context,
	erc20Abi *gethabi.ABI,
	erc20Contract gethcommon.Address,
	methodName string,
) (out string, err error) {
	res, err := k.CallContract(
		ctx, erc20Abi,
		evm.EVM_MODULE_ADDRESS,
		&erc20Contract,
		false, methodName,
	)
	if err != nil {
		return out, err
	}

	erc20Val := new(ERC20String)
	err = erc20Abi.UnpackIntoInterface(
		erc20Val, methodName, res.Ret,
	)
	if err != nil {
		return out, err
	}
	return erc20Val.Value, err
}

func (k Keeper) loadERC20Uint8(
	ctx sdk.Context,
	erc20Abi *gethabi.ABI,
	erc20Contract gethcommon.Address,
	methodName string,
) (out uint8, err error) {
	res, err := k.CallContract(
		ctx, erc20Abi,
		evm.EVM_MODULE_ADDRESS,
		&erc20Contract,
		false, methodName,
	)
	if err != nil {
		return out, err
	}

	erc20Val := new(ERC20Uint8)
	err = erc20Abi.UnpackIntoInterface(
		erc20Val, methodName, res.Ret,
	)
	if err != nil {
		return out, err
	}
	return erc20Val.Value, err
}

func (k Keeper) LoadERC20BigInt(
	ctx sdk.Context,
	abi *gethabi.ABI,
	contract gethcommon.Address,
	methodName string,
	args ...any,
) (out *big.Int, err error) {
	res, err := k.CallContract(
		ctx,
		abi,
		evm.EVM_MODULE_ADDRESS,
		&contract,
		false,
		methodName,
		args...,
	)
	if err != nil {
		return nil, err
	}

	erc20BigInt := new(ERC20BigInt)
	err = abi.UnpackIntoInterface(
		erc20BigInt, methodName, res.Ret,
	)
	if err != nil {
		return nil, err
	}

	return erc20BigInt.Value, nil
}

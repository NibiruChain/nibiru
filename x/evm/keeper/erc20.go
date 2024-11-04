// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
)

const (
	// Erc20GasLimitDeploy only used internally when deploying ERC20Minter.
	// Deployment requires ~1_600_000 gas
	Erc20GasLimitDeploy uint64 = 2_000_000
	// Erc20GasLimitQuery used only for querying name, symbol and decimals
	// Cannot be heavy. Only if the contract is malicious.
	Erc20GasLimitQuery uint64 = 100_000
	// Erc20GasLimitExecute used for transfer, mint and burn.
	// All must not exceed 200_000
	Erc20GasLimitExecute uint64 = 200_000
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
	return e.CallContract(ctx, e.ABI, from, &contract, true, Erc20GasLimitExecute, "mint", to, amount)
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
) (balanceIncrease *big.Int, resp *evm.MsgEthereumTxResponse, err error) {
	recipientBalanceBefore, err := e.BalanceOf(contract, to, ctx)
	if err != nil {
		return balanceIncrease, nil, errors.Wrap(err, "failed to retrieve recipient balance")
	}

	resp, err = e.CallContract(ctx, e.ABI, from, &contract, true, Erc20GasLimitExecute, "transfer", to, amount)
	if err != nil {
		return balanceIncrease, nil, err
	}

	var erc20Bool ERC20Bool
	err = e.ABI.UnpackIntoInterface(&erc20Bool, "transfer", resp.Ret)
	if err != nil {
		return balanceIncrease, nil, err
	}

	// Handle the case of success=false: https://github.com/NibiruChain/nibiru/issues/2080
	success := erc20Bool.Value
	if !success {
		return balanceIncrease, nil, fmt.Errorf("transfer executed but returned success=false")
	}

	recipientBalanceAfter, err := e.BalanceOf(contract, to, ctx)
	if err != nil {
		return balanceIncrease, nil, errors.Wrap(err, "failed to retrieve recipient balance")
	}

	balanceIncrease = new(big.Int).Sub(recipientBalanceAfter, recipientBalanceBefore)

	// For flexibility with fee on transfer tokens and other types of deductions,
	// we cannot assume that the amount received by the recipient is equal to
	// the call "amount". Instead, verify that the recipient got tokens and
	// return the amount.
	if balanceIncrease.Sign() <= 0 {
		return balanceIncrease, nil, fmt.Errorf(
			"amount of ERC20 tokens received MUST be positive: the balance of recipient %s would've changed by %v for token %s",
			to.Hex(), balanceIncrease.String(), contract.Hex(),
		)
	}

	return balanceIncrease, resp, err
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
	return e.CallContract(ctx, e.ABI, from, &contract, true, Erc20GasLimitExecute, "burn", amount)
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
		ctx,
		erc20Abi,
		evm.EVM_MODULE_ADDRESS,
		&erc20Contract,
		false,
		Erc20GasLimitQuery,
		methodName,
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
		false,
		Erc20GasLimitQuery,
		methodName,
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
		Erc20GasLimitQuery,
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

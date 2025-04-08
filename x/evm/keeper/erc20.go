// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"bytes"
	"fmt"
	"math"
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

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

// getCallGas returns the gas limit for a call to an ERC20 contract following 63/64 rule (EIP-150)
// protection against recursive calls ERC20 -> precompile -> ERC20.
func getCallGasWithLimit(ctx sdk.Context, gasLimit uint64) uint64 {
	availableGas := ctx.GasMeter().GasRemaining()
	callGas := availableGas - uint64(math.Floor(float64(availableGas)/64))
	return min(callGas, gasLimit)
}

// ERC20 returns a mutable reference to the keeper with an ERC20 contract ABI and
// Go functions corresponding to contract calls in the ERC20 standard like "mint"
// and "transfer" in the ERC20 standard.
func (k Keeper) ERC20() erc20Calls {
	return erc20Calls{
		Keeper: &k,
		ABI:    embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI,
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
	erc20Contract, sender, recipient gethcommon.Address, amount *big.Int,
	ctx sdk.Context, evmObj *vm.EVM,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	contractInput, err := e.ABI.Pack("mint", recipient, amount)
	if err != nil {
		return nil, err
	}
	return e.CallContractWithInput(ctx, evmObj, sender, &erc20Contract, false /*commit*/, contractInput, getCallGasWithLimit(ctx, Erc20GasLimitExecute))
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
	erc20Contract, sender, recipient gethcommon.Address, amount *big.Int,
	ctx sdk.Context, evmObj *vm.EVM,
) (balanceIncrease *big.Int, resp *evm.MsgEthereumTxResponse, err error) {
	recipientBalanceBefore, err := e.BalanceOf(erc20Contract, recipient, ctx, evmObj)
	if err != nil {
		return balanceIncrease, nil, errors.Wrap(err, "failed to retrieve recipient balance")
	}

	contractInput, err := e.ABI.Pack("transfer", recipient, amount)
	if err != nil {
		return balanceIncrease, nil, err
	}
	resp, err = e.CallContractWithInput(ctx, evmObj, sender, &erc20Contract, false /*commit*/, contractInput, getCallGasWithLimit(ctx, Erc20GasLimitExecute))
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

	recipientBalanceAfter, err := e.BalanceOf(erc20Contract, recipient, ctx, evmObj)
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
			recipient.Hex(), balanceIncrease.String(), erc20Contract.Hex(),
		)
	}

	return balanceIncrease, resp, err
}

// BalanceOf retrieves the balance of an ERC20 token for a specific account.
// Implements "ERC20.balanceOf".
func (e erc20Calls) BalanceOf(
	contract, account gethcommon.Address,
	ctx sdk.Context, evmObj *vm.EVM,
) (out *big.Int, err error) {
	return e.LoadERC20BigInt(ctx, evmObj, e.ABI, contract, "balanceOf", account)
}

/*
Burn implements "ERC20Burnable.burn"

	```solidity
	/// @dev Destroys `amount` tokens from the caller.
	function burn(uint256 amount) public virtual {
	```
*/
func (e erc20Calls) Burn(
	erc20Contract, sender gethcommon.Address, amount *big.Int,
	ctx sdk.Context, evmObj *vm.EVM,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	contractInput, err := e.ABI.Pack("burn", amount)
	if err != nil {
		return nil, err
	}
	return e.CallContractWithInput(ctx, evmObj, sender, &erc20Contract, false /*commit*/, contractInput, getCallGasWithLimit(ctx, Erc20GasLimitExecute))
}

func (e erc20Calls) LoadERC20Name(
	ctx sdk.Context, evmObj *vm.EVM, abi *gethabi.ABI, erc20 gethcommon.Address,
) (out string, err error) {
	return e.loadERC20String(ctx, evmObj, abi, erc20, "name")
}

func (e erc20Calls) LoadERC20Symbol(
	ctx sdk.Context, evmObj *vm.EVM, abi *gethabi.ABI, erc20 gethcommon.Address,
) (out string, err error) {
	return e.loadERC20String(ctx, evmObj, abi, erc20, "symbol")
}

func (e erc20Calls) LoadERC20Decimals(
	ctx sdk.Context, evmObj *vm.EVM, abi *gethabi.ABI, erc20 gethcommon.Address,
) (out uint8, err error) {
	return e.loadERC20Uint8(ctx, evmObj, abi, erc20, "decimals")
}

func (e erc20Calls) loadERC20String(
	ctx sdk.Context,
	evmObj *vm.EVM,
	erc20Abi *gethabi.ABI,
	erc20Contract gethcommon.Address,
	methodName string,
) (out string, err error) {
	input, err := erc20Abi.Pack(methodName)
	if err != nil {
		return out, err
	}
	evmResp, err := e.Keeper.CallContractWithInput(
		ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&erc20Contract,
		false,
		input,
		getCallGasWithLimit(ctx, Erc20GasLimitQuery),
	)
	if err != nil {
		return out, err
	}

	erc20Val := new(ERC20String)
	if err := erc20Abi.UnpackIntoInterface(
		erc20Val, methodName, evmResp.Ret,
	); err == nil {
		return erc20Val.Value, err
	}

	erc20Bytes32Val := new(ERC20Bytes32)
	if err := erc20Abi.UnpackIntoInterface(erc20Bytes32Val, methodName, evmResp.Ret); err == nil {
		return bytes32ToString(erc20Bytes32Val.Value), nil
	}

	return "", fmt.Errorf("failed to decode response for method %s; unable to unpack as string or bytes32", methodName)
}

func bytes32ToString(b [32]byte) string {
	return string(bytes.Trim(b[:], "\x00"))
}

func (e erc20Calls) loadERC20Uint8(
	ctx sdk.Context,
	evmObj *vm.EVM,
	erc20Abi *gethabi.ABI,
	erc20Contract gethcommon.Address,
	methodName string,
) (out uint8, err error) {
	input, err := erc20Abi.Pack(methodName)
	if err != nil {
		return out, err
	}
	evmResp, err := e.Keeper.CallContractWithInput(
		ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&erc20Contract,
		false,
		input,
		getCallGasWithLimit(ctx, Erc20GasLimitQuery),
	)
	if err != nil {
		return out, err
	}

	erc20Val := new(ERC20Uint8)
	if err := erc20Abi.UnpackIntoInterface(
		erc20Val, methodName, evmResp.Ret,
	); err == nil {
		return erc20Val.Value, err
	}

	erc20Uint256Val := new(ERC20BigInt)
	if err := erc20Abi.UnpackIntoInterface(
		erc20Uint256Val, methodName, evmResp.Ret,
	); err == nil {
		// We can safely cast to uint8 because it's nonsense for decimals to be larger than 255
		return uint8(erc20Uint256Val.Value.Uint64()), err
	}

	return 0, fmt.Errorf("failed to decode response for method %s; unable to unpack as uint8 or uint256", methodName)
}

func (e erc20Calls) LoadERC20BigInt(
	ctx sdk.Context,
	evmObj *vm.EVM,
	abi *gethabi.ABI,
	contract gethcommon.Address,
	methodName string,
	args ...any,
) (out *big.Int, err error) {
	input, err := abi.Pack(methodName, args...)
	if err != nil {
		return nil, err
	}
	evmResp, err := e.Keeper.CallContractWithInput(
		ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&contract,
		false,
		input,
		getCallGasWithLimit(ctx, Erc20GasLimitQuery),
	)
	if err != nil {
		return nil, err
	}

	erc20BigInt := new(ERC20BigInt)
	err = abi.UnpackIntoInterface(
		erc20BigInt, methodName, evmResp.Ret,
	)
	if err != nil {
		return nil, err
	}

	return erc20BigInt.Value, nil
}

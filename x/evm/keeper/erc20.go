// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	serverconfig "github.com/NibiruChain/nibiru/app/server/config"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

// FindERC20Metadata retrieves the metadata of an ERC20 token.
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - contract: The Ethereum address of the ERC20 contract.
//
// Returns:
//   - info: ERC20Metadata containing name, symbol, and decimals.
//   - err: An error if metadata retrieval fails.
func (k Keeper) FindERC20Metadata(
	ctx sdk.Context,
	contract gethcommon.Address,
) (info ERC20Metadata, err error) {
	var abi gethabi.ABI = embeds.Contract_ERC20Minter.ABI

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
	// ERC20Uint8: Unpacking type for "uint8" from Solidity. This is only used in
	// the "ERC20.decimals" function.
	ERC20Uint8  struct{ Value uint8 }
	ERC20Bool   struct{ Value bool }
	ERC20BigInt struct{ Value *big.Int }
)

// CreateFunTokenFromERC20 creates a new FunToken mapping from an existing ERC20 token.
//
// This function performs the following steps:
//  1. Checks if the ERC20 token is already registered as a FunToken.
//  2. Retrieves the metadata of the existing ERC20 token.
//  3. Verifies that the corresponding bank coin denom is not already registered.
//  4. Sets the bank coin denom metadata in the state.
//  5. Creates and inserts the new FunToken mapping.
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - erc20: The Ethereum address of the ERC20 token in HexAddr format.
//
// Returns:
//   - funtoken: The created FunToken mapping.
//   - err: An error if any step fails, nil otherwise.
//
// Possible errors:
//   - If the ERC20 token is already registered as a FunToken.
//   - If the ERC20 metadata cannot be retrieved.
//   - If the bank coin denom is already registered.
//   - If the bank metadata validation fails.
//   - If the FunToken insertion fails.
func (k *Keeper) CreateFunTokenFromERC20(
	ctx sdk.Context, erc20 eth.HexAddr,
) (funtoken evm.FunToken, err error) {
	erc20Addr := erc20.ToAddr()

	// 1 | ERC20 already registered with FunToken?
	if funtokens := k.FunTokens.Collect(
		ctx, k.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20Addr),
	); len(funtokens) > 0 {
		return funtoken, fmt.Errorf("funtoken mapping already created for ERC20 \"%s\"", erc20Addr.Hex())
	}

	// 2 | Get existing ERC20 metadata
	info, err := k.FindERC20Metadata(ctx, erc20Addr)
	if err != nil {
		return
	}
	bankDenom := fmt.Sprintf("erc20/%s", erc20.String())

	// 3 | Coin already registered with FunToken?
	_, isAlreadyCoin := k.bankKeeper.GetDenomMetaData(ctx, bankDenom)
	if isAlreadyCoin {
		return funtoken, fmt.Errorf("bank coin denom already registered with denom \"%s\"", bankDenom)
	}
	if funtokens := k.FunTokens.Collect(
		ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom),
	); len(funtokens) > 0 {
		return funtoken, fmt.Errorf("funtoken mapping already created for bank denom \"%s\"", bankDenom)
	}

	// 4 | Set bank coin denom metadata in state
	bankMetadata := bank.Metadata{
		Description: fmt.Sprintf("Bank coin representation of ERC20 token \"%s\"", erc20.String()),
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
		},
		Base:    bankDenom,
		Display: bankDenom,
		Name:    bankDenom,
		Symbol:  info.Symbol,
	}

	err = bankMetadata.Validate()
	if err != nil {
		return
	}
	k.bankKeeper.SetDenomMetaData(ctx, bankMetadata)

	// 5 | Officially create the funtoken mapping
	funtoken = evm.FunToken{
		Erc20Addr:      erc20,
		BankDenom:      bankDenom,
		IsMadeFromCoin: false,
	}

	return funtoken, k.FunTokens.SafeInsert(
		ctx, funtoken.Erc20Addr.ToAddr(),
		funtoken.BankDenom,
		funtoken.IsMadeFromCoin,
	)
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
	abi gethabi.ABI,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	commit bool,
	methodName string,
	args ...any,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	contractInput, err := abi.Pack(methodName, args...)
	if err != nil {
		err = errors.Wrap(err, "failed to pack ABI args")
		return
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
		return
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
	cfg, err := k.GetEVMConfig(
		ctx,
		ctx.BlockHeader().ProposerAddress,
		k.EthChainID(ctx),
	)
	if err != nil {
		err = fmt.Errorf("failed to load evm config: %s", err)
		return
	}
	txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash()))
	evmResp, err = k.ApplyEvmMsg(
		ctx, evmMsg, evm.NewNoOpTracer(), commit, cfg, txConfig,
	)
	if err != nil {
		return
	}

	if evmResp.Failed() {
		err = fmt.Errorf("%w: EVM error: %s", err, evmResp.VmError)
		return
	}

	return evmResp, err
}

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

	// Create a cached context for gas estimation
	cachedCtx, _ := ctx.CacheContext()

	jsonArgs, err := json.Marshal(evm.JsonTxArgs{
		From: fromAcc,
		To:   contract,
		Data: (*hexutil.Bytes)(&contractInput),
	})
	if err != nil {
		err = fmt.Errorf("failed compute gas limit to marshal tx args: %w", err)
		return
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
		err = fmt.Errorf("failed to compute gas limit: %w", err)
		return
	}

	newGasLimit = gasRes.Gas
	return newGasLimit, nil
}

func (k Keeper) LoadERC20Name(
	ctx sdk.Context, abi gethabi.ABI, erc20 gethcommon.Address,
) (out string, err error) {
	return k.LoadERC20String(ctx, abi, erc20, "name")
}

func (k Keeper) LoadERC20Symbol(
	ctx sdk.Context, abi gethabi.ABI, erc20 gethcommon.Address,
) (out string, err error) {
	return k.LoadERC20String(ctx, abi, erc20, "symbol")
}

func (k Keeper) LoadERC20Decimals(
	ctx sdk.Context, abi gethabi.ABI, erc20 gethcommon.Address,
) (out uint8, err error) {
	return k.loadERC20Uint8(ctx, abi, erc20, "decimals")
}

func (k Keeper) LoadERC20String(
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
	erc20Abi gethabi.ABI,
	erc20Contract gethcommon.Address,
	methodName string,
	args ...any,
) (out *big.Int, err error) {
	commit := false
	res, err := k.CallContract(
		ctx,
		erc20Abi,
		evm.ModuleAddressEVM(),
		&erc20Contract,
		commit,
		methodName,
		args...,
	)
	if err != nil {
		return out, err
	}

	erc20Val := new(ERC20BigInt)
	err = erc20Abi.UnpackIntoInterface(
		erc20Val, methodName, res.Ret,
	)
	if err != nil {
		return out, err
	}
	return erc20Val.Value, err
}

func (k Keeper) ERC20() Erc20Calls {
	return Erc20Calls{
		Keeper: &k,
		ABI:    embeds.Contract_ERC20Minter.ABI,
	}
}

type Erc20Calls struct {
	*Keeper
	ABI gethabi.ABI
}

func (e Erc20Calls) Mint(
	contract, from, to gethcommon.Address, amount *big.Int,
	ctx sdk.Context,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	input, err := e.ABI.Pack("mint", to, amount)
	if err != nil {
		return
	}
	commit := true
	return e.CallContractWithInput(ctx, from, &contract, commit, input)
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
func (e Erc20Calls) Transfer(
	contract, from, to gethcommon.Address, amount *big.Int,
	ctx sdk.Context,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	input, err := e.ABI.Pack("transfer", to, amount)
	if err != nil {
		return
	}
	commit := true
	return e.CallContractWithInput(ctx, from, &contract, commit, input)
}

// BalanceOf retrieves the balance of an ERC20 token for a specific account.
// Implements "ERC20.balanceOf".
func (e Erc20Calls) BalanceOf(
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
func (e Erc20Calls) Burn(
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

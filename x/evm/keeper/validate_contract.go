package keeper

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// Helper to check if the error indicates that execution was reverted
func isRevertedError(err error) bool {
	return strings.Contains(err.Error(), "execution reverted") &&
		strings.Contains(err.Error(), "unable to parse reason")
}

// HasMethodInContract checks if the contract at contractAddr
// implements the given ABI method. It constructs dummy arguments
// for the method, packs the call data (selector + arguments) and then
// calls the contract via CallContractWithInput (with commit=false).
//
// If the call fails with an error (or returns a VM error) that indicates
// the function selector was not recognized, it returns false; otherwise true.
func (k Keeper) HasMethodInContract(
	goCtx context.Context,
	contractAddr gethcommon.Address,
	method abi.Method,
) (bool, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dummyArgs := make([]interface{}, len(method.Inputs))
	for i, input := range method.Inputs {
		switch input.Type.T {
		case abi.AddressTy:
			// Use a zero address.
			dummyArgs[i] = gethcommon.Address{}
		case abi.UintTy, abi.IntTy:
			dummyArgs[i] = big.NewInt(0)
		case abi.BoolTy:
			dummyArgs[i] = false
		case abi.StringTy:
			dummyArgs[i] = ""
		default:
			// For any other type, pass nil.
			// This function has been tested mainly on ERC20 main functions,
			// so it may not work for all types.
			dummyArgs[i] = nil
		}
	}

	inputData, err := method.Inputs.Pack(dummyArgs...)
	if err != nil {
		return false, err
	}
	callData := append(method.ID, inputData...)
	const fixedGasLimit uint64 = 100000
	fromAcc := evm.EVM_MODULE_ADDRESS
	nonce := k.GetAccNonce(ctx, fromAcc)

	dummyMsg := gethcore.NewMessage(
		fromAcc,
		&contractAddr,
		nonce,
		big.NewInt(0), // value = 0
		fixedGasLimit, // use the fixed gas limit
		big.NewInt(0), // gasPrice = 0
		big.NewInt(0), // gasFeeCap = 0
		big.NewInt(0), // gasTipCap = 0
		callData,
		gethcore.AccessList{},
		false, // isFake = false
	)

	evmCfg := k.GetEVMConfig(ctx)
	txConfig := k.TxConfig(ctx, gethcommon.Hash{})
	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, txConfig)
	}
	evmObj := k.NewEVM(ctx, dummyMsg, evmCfg, nil, stateDB)

	_, err = k.CallContractWithInput(ctx, evmObj, fromAcc, &contractAddr, false, callData, fixedGasLimit)
	if err != nil {
		if isRevertedError(err) {
			return false, nil
		}
		return true, nil
	}
	return true, nil
}

// checkAllMethods ensure the contract at `contractAddr` has all the methods in `abiMethods`.
func (k Keeper) CheckAllMethods(
	ctx context.Context,
	contractAddr common.Address,
	abiMethods []abi.Method,
) error {
	for name, method := range abiMethods {
		hasMethod, err := k.HasMethodInContract(ctx, contractAddr, method)
		if err != nil {
			return err
		}
		if !hasMethod {
			return fmt.Errorf("Method %q not found in contract at %s", name, contractAddr)
		}
	}
	return nil
}

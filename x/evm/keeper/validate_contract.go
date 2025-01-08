package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// HasMethodInContract does a staticcall with the given `method`'s selector + dummy args.
// If the call reverts with something like "function selector not recognized", returns false.
//
// In your real code, this likely needs to invoke `k.evmKeeper.CallEVM` or similar.
func (k Keeper) HasMethodInContract(
	goCtx context.Context,
	contractAddr common.Address,
	method abi.Method,
) (bool, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 1. Build input (4-byte selector + encoded args).
	//    We choose dummy arguments based on the method signature.
	//    For example, if method = "balanceOf(address)", we pass a zero address or some known address.
	//    For method = "transfer(address,uint256)", pass a dummy address and zero uint256, etc.
	//
	// To illustrate, let's say we pass "0x000000000000000000000000000000000000dEaD" for addresses,
	// and 0 for all numeric arguments. This is *just* for signature detection.
	dummyArgs := make([]interface{}, len(method.Inputs))
	for i, inputDef := range method.Inputs {
		switch inputDef.Type.T {
		case abi.AddressTy:
			dummyArgs[i] = common.HexToAddress("0x000000000000000000000000000000000000dEaD")
		case abi.UintTy, abi.IntTy:
			dummyArgs[i] = 0
		default:
			dummyArgs[i] = 0
		}
	}

	input, err := method.Inputs.Pack(dummyArgs...)
	if err != nil {
		return false, fmt.Errorf("packing dummy args: %w", err)
	}

	// Prepend the 4-byte method selector
	sig := method.ID
	callData := append(sig, input...)

	// 2. Make a call message
	callMsg := evm.JsonTxArgs{
		From:  &common.Address{},
		To:    &contractAddr,
		Input: (*hexutil.Bytes)(&callData),
	}

	jsonTxArgs, err := json.Marshal(&callMsg)
	if err != nil {
		return false, fmt.Errorf("marshaling call message: %w", err)
	}

	ethCallRequest := evm.EthCallRequest{
		Args:            jsonTxArgs,
		GasCap:          gethparams.TxGas,
		ProposerAddress: sdk.ConsAddress(ctx.BlockHeader().ProposerAddress),
		ChainId:         k.EthChainID(ctx).Int64(),
	}

	_, err = k.EstimateGasForEvmCallType(goCtx, &ethCallRequest, evm.CallTypeRPC)

	// Distinguish an error that indicates "function not found" vs. a normal revert
	if strings.Contains(err.Error(), "invalid opcode") || strings.Contains(err.Error(), "does not exist") {
		return false, nil
	}

	// 4. If we got a "function not found" style revert, return false.
	if strings.Contains(err.Error(), "selector not recognized") {
		return false, nil
	}

	fmt.Println("err: ", err)

	// If we arrive here, that means the call at least recognized the function signature
	// (the contract may revert for other reasons, but the method "exists").
	return true, nil
}

// checkAllMethods ensure the contract at `contractAddr` has all the methods in `abiMethods`.
func (k Keeper) CheckAllethods(
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

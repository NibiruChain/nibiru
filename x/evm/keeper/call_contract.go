package keeper

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// CallContract invokes a smart contract with the given [contractInput]
// or deploys a new contract.
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - evmObj: EVM instance carrying the current StateDB  and interpreter.
//   - fromAcc: The Ethereum address of the account initiating the contract call.
//   - contract: Pointer to the Ethereum address of the contract. Nil if new
//     contract is deployed.
//   - contractInput: Hexadecimal-encoded bytes used as input data to the call.
//   - gasLimit: Maximum gas available for execution.
//   - commit: Boolean for whether to commit the [vm.StateDB]. This functions
//     handles both contract method calls and simulations, depending on the
//     `commit` parameter.
//   - weiValue: wei value to transfer with the call. Giving `nil` means 0.
//
// Returns:
//   - evmResp: Execution result containing  gas usage, return data, logs, and VM
//     Errors.
//   - err: Error with
func (k Keeper) CallContract(
	ctx sdk.Context,
	evmObj *vm.EVM,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	contractInput []byte,
	gasLimit uint64,
	commit bool,
	weiValue *big.Int,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	nonce := k.GetAccNonce(ctx, fromAcc)

	unusedBigInt := big.NewInt(0)
	if weiValue == nil {
		weiValue = unusedBigInt
	}
	evmMsg := core.Message{
		To:               contract,
		From:             fromAcc,
		Nonce:            nonce,
		Value:            weiValue, // amount
		GasLimit:         gasLimit,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             contractInput,
		AccessList:       gethcore.AccessList{},
		BlobGasFeeCap:    &big.Int{},
		BlobHashes:       []gethcommon.Hash{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}

	// Generating TxConfig with an empty tx hash as there is no actual eth tx
	// sent by a user
	txConfig := k.TxConfig(ctx, gethcommon.BigToHash(big.NewInt(0)))

	var applyErr error
	evmResp, applyErr = k.ApplyEvmMsg(
		ctx, evmMsg, evmObj, commit /*commit*/, txConfig.TxHash,
	)
	if applyErr != nil {
		ctx.WithLastErrApplyEvmMsg(applyErr)
		return nil, applyErr
	}

	if evmResp != nil {
		gasErr := evm.SafeConsumeGas(ctx, evmResp.GasUsed, "CallContract")
		if gasErr != nil {
			return nil, gasErr
		}
	}

	if evmResp != nil && evmResp.Failed() {
		if lastEvmErr := ctx.LastErrApplyEvmMsg(); lastEvmErr != nil {
			evmResp.VmError += ": " + lastEvmErr.Error()
		}
		if strings.Contains(evmResp.VmError, vm.ErrOutOfGas.Error()) {
			err = fmt.Errorf(
				"VMError: %s: gas required exceeds gas limit (%d)",
				evmResp.VmError, gasLimit,
			)
			return
		}
		if evmResp.VmError == vm.ErrExecutionReverted.Error() {
			err = fmt.Errorf(
				"VMError: %s",
				evm.NewRevertError(evmResp.Ret),
			)
			return
		}
		err = fmt.Errorf("VMError: %s", evmResp.VmError)
	}

	return evmResp, err
}

package keeper

import (
	"fmt"
	"math/big"
	"strings"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/evm"
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
// depending on the 'commit' parameter.
func (k Keeper) CallContract(
	ctx sdk.Context,
	abi *gethabi.ABI,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	commit bool,
	gasLimit uint64,
	methodName string,
	args ...any,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	contractInput, err := abi.Pack(methodName, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack ABI args: %w", err)
	}
	evmResp, _, err = k.CallContractWithInput(ctx, fromAcc, contract, commit, contractInput, gasLimit)
	return evmResp, err
}

// CallContractWithInput invokes a smart contract with the given [contractInput]
// or deploys a new contract.
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - fromAcc: The Ethereum address of the account initiating the contract call.
//   - contract: Pointer to the Ethereum address of the contract. Nil if new
//     contract is deployed.
//   - commit: Boolean flag indicating whether to commit the transaction (true)
//     or simulate it (false).
//   - contractInput: Hexadecimal-encoded bytes use as input data to the call.
//
// Note: This function handles both contract method calls and simulations,
// depending on the 'commit' parameter. It uses a default gas limit.
func (k Keeper) CallContractWithInput(
	ctx sdk.Context,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	commit bool,
	contractInput []byte,
	gasLimit uint64,
) (evmResp *evm.MsgEthereumTxResponse, evmObj *vm.EVM, err error) {
	// This is a `defer` pattern to add behavior that runs in the case that the
	// error is non-nil, creating a concise way to add extra information.
	defer HandleOutOfGasPanic(&err, "CallContractError")
	nonce := k.GetAccNonce(ctx, fromAcc)

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
		err = errors.Wrapf(err, "failed to load EVM config")
		return
	}

	// Generating TxConfig with an empty tx hash as there is no actual eth tx
	// sent by a user
	txConfig := k.TxConfig(ctx, gethcommon.BigToHash(big.NewInt(0)))

	evmResp, evmObj, err = k.ApplyEvmMsg(
		ctx, evmMsg, evm.NewNoOpTracer(), commit, evmCfg, txConfig, true,
	)
	if err != nil {
		// We don't know the actual gas used, so consuming the gas limit
		k.ResetGasMeterAndConsumeGas(ctx, gasLimit)
		err = errors.Wrap(err, "failed to apply ethereum core message")
		return
	}

	if evmResp.Failed() {
		k.ResetGasMeterAndConsumeGas(ctx, evmResp.GasUsed)
		if strings.Contains(evmResp.VmError, vm.ErrOutOfGas.Error()) {
			err = fmt.Errorf("gas required exceeds allowance (%d)", gasLimit)
			return
		}
		if evmResp.VmError == vm.ErrExecutionReverted.Error() {
			err = fmt.Errorf("VMError: %w", evm.NewExecErrorWithReason(evmResp.Ret))
			return
		}
		err = fmt.Errorf("VMError: %s", evmResp.VmError)
		return
	}

	// Success, update block gas used and bloom filter
	if commit {
		blockGasUsed, err := k.AddToBlockGasUsed(ctx, evmResp.GasUsed)
		if err != nil {
			k.ResetGasMeterAndConsumeGas(ctx, ctx.GasMeter().Limit())
			return nil, nil, errors.Wrap(err, "error adding transient gas used to block")
		}
		k.ResetGasMeterAndConsumeGas(ctx, blockGasUsed)
		k.updateBlockBloom(ctx, evmResp, uint64(txConfig.LogIndex))
		// TODO: remove after migrating logs
		//err = k.EmitLogEvents(ctx, evmResp)
		//if err != nil {
		//	return nil, nil, errors.Wrap(err, "error emitting tx logs")
		//}

		// blockTxIdx := uint64(txConfig.TxIndex) + 1
		// k.EvmState.BlockTxIndex.Set(ctx, blockTxIdx)
	}
	return evmResp, evmObj, nil
}

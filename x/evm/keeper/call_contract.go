package keeper

import (
	"fmt"
	"math/big"
	"strings"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

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
	evmObj *vm.EVM,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	commit bool,
	contractInput []byte,
	gasLimit uint64,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
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

	// Generating TxConfig with an empty tx hash as there is no actual eth tx
	// sent by a user
	txConfig := k.TxConfig(ctx, gethcommon.BigToHash(big.NewInt(0)))
	evmResp, err = k.ApplyEvmMsg(
		ctx, evmMsg, evmObj, evm.NewNoOpTracer(), commit, txConfig.TxHash, true,
	)
	if err != nil {
		err = errors.Wrap(err, "failed to apply ethereum core message")
		return
	}

	if evmResp.Failed() {
		if strings.Contains(evmResp.VmError, vm.ErrOutOfGas.Error()) {
			err = fmt.Errorf("gas required exceeds allowance (%d)", gasLimit)
			return
		}
		if evmResp.VmError == vm.ErrExecutionReverted.Error() {
			err = fmt.Errorf("VMError: %w", evm.NewRevertError(evmResp.Ret))
			return
		}
		err = fmt.Errorf("VMError: %s", evmResp.VmError)
		return
	}

	// Success, update block gas used and bloom filter
	if commit {
		k.updateBlockBloom(ctx, evmResp, uint64(txConfig.LogIndex))
		// TODO: remove after migrating logs
		//err = k.EmitLogEvents(ctx, evmResp)
		//if err != nil {
		//	return nil, nil, errors.Wrap(err, "error emitting tx logs")
		//}

		// blockTxIdx := uint64(txConfig.TxIndex) + 1
		// k.EvmState.BlockTxIndex.Set(ctx, blockTxIdx)
	}
	return evmResp, nil
}

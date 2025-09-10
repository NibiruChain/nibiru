package keeper

import (
	"fmt"
	"math/big"
	"strings"

	sdkioerrors "cosmossdk.io/errors"
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
//   - fromAcc: The Ethereum address of the account initiating the contract call.
//   - contract: Pointer to the Ethereum address of the contract. Nil if new
//     contract is deployed.
//   - contractInput: Hexadecimal-encoded bytes use as input data to the call.
//
// Note: This function handles both contract method calls and simulations,
// depending on the 'commit' parameter. It uses a default gas limit.
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
	// This is a `defer` pattern to add behavior that runs in the case that the
	// error is non-nil, creating a concise way to add extra information.
	defer HandleOutOfGasPanic(&err, "CallContractError")()
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
	evmResp, err = k.ApplyEvmMsg(
		ctx, evmMsg, evmObj, commit /*commit*/, txConfig.TxHash,
	)
	if evmResp != nil {
		ctx.GasMeter().ConsumeGas(evmResp.GasUsed, "CallContractWithInput")
	}
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to apply ethereum core message")
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
	return evmResp, nil
}

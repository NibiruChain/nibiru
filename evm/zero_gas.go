package evm

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/core"
)

// DefaultZeroGasTxPriority is the fixed mempool priority for classified
// zero-gas EVM transactions. Raw wallet fee fields are ignored for payment and
// must not buy priority.
const DefaultZeroGasTxPriority int64 = 0

// IsZeroGasTxData returns true when txData targets a contract in the
// `always_zero_gas_contracts` set.
func IsZeroGasTxData(
	ctx sdk.Context,
	sudoKeeper SudoKeeper,
	txData TxData,
) bool {
	if sudoKeeper == nil || txData == nil {
		return false
	}

	to := txData.GetTo()
	if to == nil {
		return false
	}

	contracts := sudoKeeper.GetZeroGasEvmContracts(ctx)
	if len(contracts) == 0 {
		return false
	}

	_, ok := contracts[*to]
	return ok
}

// IsZeroGasMsgEthereumTx classifies a signed EVM message without mutating the
// raw signed transaction data.
func IsZeroGasMsgEthereumTx(
	ctx sdk.Context,
	sudoKeeper SudoKeeper,
	msg *MsgEthereumTx,
) (bool, TxData, error) {
	if msg == nil {
		return false, nil, fmt.Errorf("nil MsgEthereumTx")
	}

	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return false, nil, err
	}

	return IsZeroGasTxData(ctx, sudoKeeper, txData), txData, nil
}

// IsZeroGasJsonTxArgs classifies JSON-RPC call args using the same destination
// policy as signed execution.
func IsZeroGasJsonTxArgs(
	ctx sdk.Context,
	sudoKeeper SudoKeeper,
	args JsonTxArgs,
) bool {
	if sudoKeeper == nil || args.To == nil {
		return false
	}

	contracts := sudoKeeper.GetZeroGasEvmContracts(ctx)
	if len(contracts) == 0 {
		return false
	}

	_, ok := contracts[*args.To]
	return ok
}

// WithZeroGasMeta stores the existing zero-gas marker on ctx.
func WithZeroGasMeta(ctx sdk.Context) sdk.Context {
	return ctx.WithValue(CtxKeyZeroGasMeta, &ZeroGasMeta{})
}

// NormalizeZeroGasMessage returns a copy of msg with fee-price semantics zeroed.
// It must be called only after sender recovery for signed transactions.
func NormalizeZeroGasMessage(msg core.Message) core.Message {
	msg.GasPrice = new(big.Int)
	msg.GasFeeCap = new(big.Int)
	msg.GasTipCap = new(big.Int)
	return msg
}

// ValidateZeroGasTxData preserves non-fee stateless checks for classified
// zero-gas transactions.
func ValidateZeroGasTxData(txData TxData) error {
	if txData.GetTo() == nil {
		return sdkioerrors.Wrap(sdkerrors.ErrInvalidRequest, "zero-gas tx must not be contract creation")
	}
	for _, err := range []error{
		ValidateTxDataAmount(txData),
		ValidateTxDataTo(txData),
		ValidateTxDataChainID(txData),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

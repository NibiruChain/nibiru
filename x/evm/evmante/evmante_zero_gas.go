package evmante

// Copyright (c) 2023-2024 Nibi, Inc.
//
// evmante_zero_gas.go: Ante step for detecting zero-gas EVM transactions.

import (
	sdkioerrors "cosmossdk.io/errors"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

var _ AnteStep = AnteStepDetectZeroGas

// AnteStepDetectZeroGas detects eligible zero-gas EVM txs and sets a context
// marker so subsequent ante steps (VerifyEthAcc, DeductGas) and the msg_server
// can skip balance checks and fee deduction/refund; CanTransfer runs for structural validity.
//
// Eligibility:
//   - tx.To is in ZeroGasActors.AlwaysZeroGasContracts (EVM hex address list)
//   - tx.Value == 0
//
// No state mutations. Only sets ZeroGasMeta in context as a marker.
func AnteStepDetectZeroGas(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return sdkioerrors.Wrap(err, "AnteStepDetectZeroGas: failed to unpack tx data")
	}

	to := txData.GetTo()
	if to == nil {
		return nil
	}

	evmContracts := k.SudoKeeper.GetZeroGasEvmContracts(sdb.Ctx())
	if len(evmContracts) == 0 {
		return nil
	}

	matchesZeroGasContract := false
	for _, eip55Addr := range evmContracts {
		if eip55Addr.Address == *to {
			matchesZeroGasContract = true
			break
		}
	}
	if !matchesZeroGasContract {
		return nil
	}

	if txValue := txData.GetValueWei(); txValue != nil && txValue.Sign() != 0 {
		return nil
	}

	sdb.SetCtx(
		sdb.Ctx().WithValue(evm.CtxKeyZeroGasMeta, &evm.ZeroGasMeta{}),
	)

	return nil
}

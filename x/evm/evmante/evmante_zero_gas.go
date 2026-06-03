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
	isZeroGas, _, err := evm.IsZeroGasMsgEthereumTx(sdb.Ctx(), k.SudoKeeper, msgEthTx)
	if err != nil {
		return sdkioerrors.Wrap(err, "AnteStepDetectZeroGas: failed to unpack tx data")
	}
	if !isZeroGas {
		return nil
	}

	sdb.SetCtx(evm.WithZeroGasMeta(sdb.Ctx()))

	return nil
}

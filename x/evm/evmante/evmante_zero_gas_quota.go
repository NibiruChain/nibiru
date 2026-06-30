package evmante

// Copyright (c) 2023-2024 Nibi, Inc.
//
// evmante_zero_gas_quota.go: Per-block quota enforcement for zero-gas EVM txs.

import (
	sdkioerrors "cosmossdk.io/errors"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

var _ AnteStep = AnteStepZeroGasBlockQuota

// maxZeroGasTxsPerBlock is a consensus-critical constant.
//
// Set to 0 to disable quota enforcement.
//
// Note: For production, making this governance-controlled (e.g. via x/sudo state
// or x/evm params) is preferable to a hardcoded constant.
const maxZeroGasTxsPerBlock uint64 = 100

// AnteStepZeroGasBlockQuota enforces a per-block cap on the number of zero-gas
// EVM transactions that can be included. It uses the EVM transient store, which
// is reset at Commit (end of block).
//
// Rationale: Zero-gas txs bypass fee-based spam controls. A quota provides a
// deterministic backstop against free blockspace consumption.
//
// Enforcement:
//   - DeliverTx only. (CheckTx/mempool admission is intentionally unaffected.)
func AnteStepZeroGasBlockQuota(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	if maxZeroGasTxsPerBlock == 0 {
		return nil
	}
	if !evmstate.IsDeliverTx(sdb.Ctx()) {
		return nil
	}
	if !evm.IsZeroGasEthTx(sdb.Ctx()) {
		return nil
	}

	count := k.EvmState.BlockZeroGasTxCount.GetOr(sdb.Ctx(), 0)
	if count >= maxZeroGasTxsPerBlock {
		return sdkioerrors.Wrapf(
			evm.ErrZeroGasBlockQuotaExceeded,
			"max=%d", maxZeroGasTxsPerBlock,
		)
	}

	k.EvmState.BlockZeroGasTxCount.Set(sdb.Ctx(), count+1)
	return nil
}

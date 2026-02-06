package evmante

// Copyright (c) 2023-2024 Nibi, Inc.
//
// evmante_zero_gas.go: Ante step for crediting zero-gas EVM transactions.

import (
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/ethereum/go-ethereum/core/tracing"
)

var _ AnteStep = AnteStepCreditZeroGas

// AnteStepCreditZeroGas detects eligible zero-gas EVM txs and temporarily
// credits the sender with a fixed balance so that subsequent ante checks
// (VerifyEthAcc, CanTransfer, DeductGas) can run unchanged.
//
// Eligibility:
//   - tx.To is in ZeroGasActors.AlwaysZeroGasContracts (EVM hex address list)
//   - tx.Value == 0
//
// Credit semantics:
//   - Uses a fixed credit amount of 10 NIBI (in wei), not txData.Cost().
//   - Mutates balances unconditionally for now; TODO: handle CheckTx/ReCheckTx/
//     simulation phases explicitly if needed.
//   - Stores ZeroGasMeta in context so downstream logic and msg_server can
//     identify zero-gas txs. The getter returns nil when not a zero-gas tx.
func AnteStepCreditZeroGas(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	// Unpack tx data (gas, value, to, etc.)
	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return sdkioerrors.Wrap(err, "AnteStepCreditZeroGas: failed to unpack tx data")
	}

	to := txData.GetTo()
	// Contract deploys (To == nil) are not eligible.
	if to == nil {
		return nil
	}

	// Load zero-gas EVM contracts from the Sudo keeper via the EVM keeper.
	evmContracts := k.SudoKeeper.GetZeroGasEvmContracts(sdb.Ctx())
	if len(evmContracts) == 0 {
		return nil
	}

	// Require: The `to` addr must be in the configured zero-gas EVM contracts
	{
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
	}
	// Require: // Sponsored txs for this mechanism must have zero value.
	if txValue := txData.GetValueWei(); txValue != nil && txValue.Sign() != 0 {
		return nil
	}

	// Compute fixed credit amount: 10 NIBI in wei.
	creditWeiBig := evm.NativeToWei(
		new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000)),
	)
	if creditWeiBig.Sign() <= 0 {
		// No credit needed if computed amount is non-positive.
		return nil
	}

	// TODO(UD-DEBUG): Respect execution phase (CheckTx/ReCheckTx/Simulate vs DeliverTx)
	// and ensure that balance mutation happens only where appropriate.
	// For now we always mutate balance here and rely on future work to refine this.

	// Apply credit to sender balance.
	fromAddr := msgEthTx.FromAddr()
	creditWeiU256 := uint256.MustFromBig(creditWeiBig)
	sdb.AddBalance(fromAddr, creditWeiU256, tracing.BalanceChangeTransfer)

	// Set or update ZeroGasMeta in context.
	meta := evm.GetZeroGasMeta(sdb.Ctx())
	if meta == nil {
		meta = &evm.ZeroGasMeta{}
	}
	// Only set CreditedWei if it has not been set yet.
	if meta.CreditedWei == nil || meta.CreditedWei.Sign() == 0 {
		meta.CreditedWei = new(big.Int).Set(creditWeiBig)
	}
	meta.Phase = evm.ZeroGasPhaseCredited

	sdb.SetCtx(
		sdb.Ctx().
			WithValue(evm.CtxKeyZeroGasMeta, meta),
	)

	return nil
}

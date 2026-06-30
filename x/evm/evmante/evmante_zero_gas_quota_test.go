package evmante_test

import (
	"math/big"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func TestAnteStepZeroGasBlockQuota_PerBlockTxCount(t *testing.T) {
	deps := evmtest.NewTestDeps()

	// Configure ZeroGasActors with an always_zero_gas_contracts entry.
	targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{targetAddr.Hex()},
	})

	// Create a tx that targets the allowlisted contract with zero value.
	to := targetAddr
	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:    0,
		GasLimit: 50_000,
		GasPrice: big.NewInt(0),
		To:       &to,
		Amount:   big.NewInt(0),
	})
	tx.From = deps.Sender.EthAddr.Hex()

	// Fill the quota within the same block.
	for i := uint64(0); i < 100; i++ {
		sdb := deps.NewStateDB()
		err := evmante.AnteStepDetectZeroGas(
			sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED,
		)
		require.NoError(t, err)
		require.True(t, evm.IsZeroGasEthTx(sdb.Ctx()))

		err = evmante.AnteStepZeroGasBlockQuota(
			sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED,
		)
		require.NoError(t, err)
		deps.Commit()
	}

	// The next zero-gas tx in the same block should fail.
	{
		sdb := deps.NewStateDB()
		err := evmante.AnteStepDetectZeroGas(
			sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED,
		)
		require.NoError(t, err)
		require.True(t, evm.IsZeroGasEthTx(sdb.Ctx()))

		err = evmante.AnteStepZeroGasBlockQuota(
			sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED,
		)
		require.Error(t, err)
		require.ErrorContains(t, err, "zero-gas block quota exceeded")
	}
}

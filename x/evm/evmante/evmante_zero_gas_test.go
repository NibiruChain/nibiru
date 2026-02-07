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

var (
	addr1 = gethcommon.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 = gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	addr3 = gethcommon.HexToAddress("0x3333333333333333333333333333333333333333")
	addr4 = gethcommon.HexToAddress("0x4444444444444444444444444444444444444444")
)

func TestAnteStepCreditZeroGas_NonEligible_NoMetaNoCredit(t *testing.T) {
	deps := evmtest.NewTestDeps()

	// Ensure ZeroGasActors is empty so no tx is eligible.
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.DefaultZeroGasActors())

	sdb := deps.NewStateDB()

	// Minimal MsgEthereumTx with a non-allowlisted To address.
	toAddr := addr1
	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:    0,
		GasLimit: 21_000,
		GasPrice: big.NewInt(1),
		To:       &toAddr,
		Amount:   big.NewInt(0),
	})
	tx.From = deps.Sender.EthAddr.Hex()

	// Pre-balance snapshot.
	from := tx.FromAddr()
	initialBal := new(big.Int).Set(sdb.GetBalance(from).ToBig())

	err := evmante.AnteStepCreditZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	// No meta should be set.
	meta := evm.GetZeroGasMeta(sdb.Ctx())
	require.Nil(t, meta)

	// Balance should be unchanged.
	require.Equal(t, 0, initialBal.Cmp(sdb.GetBalance(from).ToBig()))
}

func TestAnteStepCreditZeroGas_Eligible_SetsMetaAndCreditsBalance(t *testing.T) {
	deps := evmtest.NewTestDeps()

	// Configure ZeroGasActors with an always_zero_gas_contracts entry.
	targetAddr := addr2
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{targetAddr.Hex()},
	})

	sdb := deps.NewStateDB()

	// Create a tx that targets the allowlisted contract with zero value.
	to := targetAddr
	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:    0,
		GasLimit: 50_000,
		GasPrice: big.NewInt(1),
		To:       &to,
		Amount:   big.NewInt(0),
	})
	tx.From = deps.Sender.EthAddr.Hex()

	from := tx.FromAddr()
	initialBal := new(big.Int).Set(sdb.GetBalance(from).ToBig())

	err := evmante.AnteStepCreditZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	meta := evm.GetZeroGasMeta(sdb.Ctx())
	require.NotNil(t, meta)
	require.NotNil(t, meta.CreditedWei)

	// Credited balance should be greater than the initial balance.
	require.Greater(t, sdb.GetBalance(from).ToBig().Cmp(initialBal), 0)
}

func TestAnteStepCreditZeroGas_Eligible_NonZeroValue_NoCredit(t *testing.T) {
	deps := evmtest.NewTestDeps()
	// Configure ZeroGasActors with an always_zero_gas_contracts entry.
	targetAddr := addr3
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{targetAddr.Hex()},
	})

	sdb := deps.NewStateDB()

	// Create a tx that targets the allowlisted contract but with non-zero value.
	to := targetAddr
	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:    0,
		GasLimit: 50_000,
		GasPrice: big.NewInt(1),
		To:       &to,
		Amount:   big.NewInt(1), // non-zero value should make tx ineligible
	})
	tx.From = deps.Sender.EthAddr.Hex()

	from := tx.FromAddr()
	initialBal := new(big.Int).Set(sdb.GetBalance(from).ToBig())

	err := evmante.AnteStepCreditZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	// No meta should be set for non-zero value.
	meta := evm.GetZeroGasMeta(sdb.Ctx())
	require.Nil(t, meta)

	// Balance should be unchanged.
	require.Equal(t, 0, initialBal.Cmp(sdb.GetBalance(from).ToBig()))
}

func TestAnteStepCreditZeroGas_SetsMetaInContextWhenCheckTx(t *testing.T) {
	deps := evmtest.NewTestDeps()

	// Emulate CheckTx context.
	deps.SetCtx(deps.Ctx().WithIsCheckTx(true))

	// Allowlist a contract.
	targetAddr := addr4
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{targetAddr.Hex()},
	})

	sdb := deps.NewStateDB()

	to := targetAddr
	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:    0,
		GasLimit: 50_000,
		GasPrice: big.NewInt(1),
		To:       &to,
		Amount:   big.NewInt(0),
	})
	tx.From = deps.Sender.EthAddr.Hex()

	from := tx.FromAddr()
	initialBal := new(big.Int).Set(sdb.GetBalance(from).ToBig())

	err := evmante.AnteStepCreditZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	// Meta should be present even in CheckTx.
	meta := evm.GetZeroGasMeta(sdb.Ctx())
	require.NotNil(t, meta)
	require.NotNil(t, meta.CreditedWei)

	// For now we also mutate balance in CheckTx (see TODO in implementation).
	// Just assert that the credited amount is non-zero and balance increased.
	require.True(t, meta.CreditedWei.Sign() > 0)
	require.Equal(t, -1, initialBal.Cmp(sdb.GetBalance(from).ToBig()))
}

func TestAnteStepDeductGas_SetsPaidWeiForZeroGasTx(t *testing.T) {
	deps := evmtest.NewTestDeps()

	// Allowlist a contract for zero-gas.
	targetAddr := addr1
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{targetAddr.Hex()},
	})

	sdb := deps.NewStateDB()

	to := targetAddr
	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:    0,
		GasLimit: 50_000,
		GasPrice: big.NewInt(1),
		To:       &to,
		Amount:   big.NewInt(0),
	})
	tx.From = deps.Sender.EthAddr.Hex()

	// First run credit step to populate meta.
	err := evmante.AnteStepCreditZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	// Then run deduct gas.
	err = evmante.AnteStepDeductGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	meta := evm.GetZeroGasMeta(sdb.Ctx())
	require.NotNil(t, meta)
	require.NotNil(t, meta.PaidWei)
	require.True(t, meta.PaidWei.Sign() > 0)
}

func TestAnteStepDeductGas_DoesNotSetMetaForNormalTx(t *testing.T) {
	deps := evmtest.NewTestDeps()

	// Ensure ZeroGasActors is empty so no tx is eligible.
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.DefaultZeroGasActors())

	sdb := deps.NewStateDB()

	toAddr := addr2
	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:    0,
		GasLimit: 21_000,
		GasPrice: big.NewInt(1),
		To:       &toAddr,
		Amount:   big.NewInt(0),
	})
	tx.From = deps.Sender.EthAddr.Hex()

	// Run deduct gas directly with no prior crediting or funding; this will fail
	// due to insufficient funds, but we still assert that no zero-gas meta is set.
	err := evmante.AnteStepDeductGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.Error(t, err)

	meta := evm.GetZeroGasMeta(sdb.Ctx())
	require.Nil(t, meta)
}

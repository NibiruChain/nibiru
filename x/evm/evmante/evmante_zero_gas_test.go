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

func TestAnteStepDetectZeroGas_NonEligible_NoMeta(t *testing.T) {
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

	err := evmante.AnteStepDetectZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	// No meta should be set.
	require.False(t, evm.IsZeroGasEthTx(sdb.Ctx()))

	// Balance should be unchanged (no credit in bypass approach).
	require.Equal(t, 0, initialBal.Cmp(sdb.GetBalance(from).ToBig()))
}

func TestAnteStepDetectZeroGas_Eligible_SetsMetaNoCredit(t *testing.T) {
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

	err := evmante.AnteStepDetectZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	require.True(t, evm.IsZeroGasEthTx(sdb.Ctx()), "ZeroGasMeta should be set for eligible zero-gas tx")

	// Bypass approach: no balance mutation. Balance should be unchanged.
	require.Equal(t, 0, initialBal.Cmp(sdb.GetBalance(from).ToBig()))
}

func TestAnteStepDetectZeroGas_Eligible_NonZeroValue_NoMeta(t *testing.T) {
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

	err := evmante.AnteStepDetectZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	// No meta should be set for non-zero value.
	require.False(t, evm.IsZeroGasEthTx(sdb.Ctx()))

	// Balance should be unchanged.
	require.Equal(t, 0, initialBal.Cmp(sdb.GetBalance(from).ToBig()))
}

func TestAnteStepDetectZeroGas_SetsMetaInContextWhenCheckTx(t *testing.T) {
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

	err := evmante.AnteStepDetectZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	// Meta should be present even in CheckTx.
	require.True(t, evm.IsZeroGasEthTx(sdb.Ctx()))

	// Bypass approach: no balance mutation.
	require.Equal(t, 0, initialBal.Cmp(sdb.GetBalance(from).ToBig()))
}

func TestAnteStepDeductGas_SkipsForZeroGasTx(t *testing.T) {
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

	// Run detect step to populate meta.
	err := evmante.AnteStepDetectZeroGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	from := tx.FromAddr()
	balBefore := new(big.Int).Set(sdb.GetBalance(from).ToBig())

	// DeductGas should skip for zero-gas tx (return nil, no deduction).
	err = evmante.AnteStepDeductGas(
		sdb,
		sdb.Keeper(),
		tx,
		false,
		ANTE_OPTIONS_UNUSED,
	)
	require.NoError(t, err)

	// Balance should be unchanged (no deduction).
	require.Equal(t, 0, balBefore.Cmp(sdb.GetBalance(from).ToBig()))
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

	require.False(t, evm.IsZeroGasEthTx(sdb.Ctx()))
}

func TestVerifyEthAcc_ZeroGas_CreatesAccountWhenMissing(t *testing.T) {
	deps := evmtest.NewTestDeps()

	targetAddr := addr2
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{targetAddr.Hex()},
	})

	sdb := deps.NewStateDB()

	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:    0,
		GasLimit: 50_000,
		GasPrice: big.NewInt(0),
		To:       &targetAddr,
		Amount:   big.NewInt(0),
	})
	tx.From = deps.Sender.EthAddr.Hex()

	err := evmante.AnteStepDetectZeroGas(sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED)
	require.NoError(t, err)
	require.True(t, evm.IsZeroGasEthTx(sdb.Ctx()))

	err = evmante.AnteStepVerifyEthAcc(sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED)
	require.NoError(t, err)

	acc := sdb.Keeper().GetAccount(sdb.Ctx(), tx.FromAddr())
	require.NotNil(t, acc, "VerifyEthAcc must create sender account when missing for zero-gas tx")
}

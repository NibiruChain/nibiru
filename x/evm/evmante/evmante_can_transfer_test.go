package evmante_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func (s *Suite) TestCanTransfer() {
	testCases := []AnteTC{
		{
			Name:        "happy: signed tx, sufficient funds",
			EvmAnteStep: evmante.AnteStepCanTransfer,
			TxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx {
				s.NoError(
					testapp.FundAccount(
						deps.App.BankKeeper,
						deps.Ctx(),
						deps.Sender.NibiruAddr,
						sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 100)),
					),
				)

				txMsg := evmtest.HappyTransferTx(deps, 0)
				txBuilder := deps.App.GetTxConfig().NewTxBuilder()

				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
				err := txMsg.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)

				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)

				evmTx, err := evm.RequireStandardEVMTxMsg(tx)
				s.NoError(err)
				return evmTx
			},
			WantErr: "",
		},
		{
			Name:        "sad: signed tx, insufficient funds",
			EvmAnteStep: evmante.AnteStepCanTransfer,
			TxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx {
				txMsg := evmtest.HappyTransferTx(deps, 0)
				txBuilder := deps.App.GetTxConfig().NewTxBuilder()

				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
				err := txMsg.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)

				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)

				evmTx, err := evm.RequireStandardEVMTxMsg(tx)
				s.NoError(err)
				return evmTx
			},
			WantErr: "insufficient funds",
		},
		{
			Name:        "sad: unsigned tx",
			EvmAnteStep: evmante.AnteStepCanTransfer,
			TxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx {
				txMsg := evmtest.HappyTransferTx(deps, 0)
				txBuilder := deps.App.GetTxConfig().NewTxBuilder()

				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)

				evmTx, err := evm.RequireStandardEVMTxMsg(tx)
				s.NoError(err)
				return evmTx
			},
			WantErr: "invalid transaction",
		},
	}

	RunAnteTCs(&s.Suite, testCases)
}

func (s *Suite) TestVerifyEthAcc() {
	testCases := []struct {
		name               string
		beforeTxSetup      func(deps *evmtest.TestDeps, sdb *evmstate.SDB)
		txSetup            func(deps *evmtest.TestDeps) *evm.MsgEthereumTx
		wantErr            string
		zeroGasEligible    bool // if true, run DetectZeroGas before VerifyEthAcc and assert account exists after
	}{
		{
			name:          "happy: sender with funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
				AddBalanceSigned(sdb, deps.Sender.EthAddr, evm.NativeToWei(happyGasLimit()))
			},
			txSetup:         evmtest.HappyCreateContractTx,
			wantErr:         "",
			zeroGasEligible: false,
		},
		{
			name:            "sad: sender has insufficient gas balance",
			beforeTxSetup:   func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {},
			txSetup:         evmtest.HappyCreateContractTx,
			wantErr:         "sender balance < tx cost",
			zeroGasEligible: false,
		},
		{
			name:            "sad: invalid tx",
			beforeTxSetup:   func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {},
			txSetup:         func(deps *evmtest.TestDeps) *evm.MsgEthereumTx { return new(evm.MsgEthereumTx) },
			wantErr:         "failed to unpack tx data",
			zeroGasEligible: false,
		},
		{
			name: "sad: empty from addr",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {},
			txSetup: func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps)
				tx.From = ""
				return tx
			},
			wantErr:         "from address cannot be empty",
			zeroGasEligible: false,
		},
		{
			name: "zero-gas: sender has no balance, account created if missing, VerifyEthAcc passes",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
				targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
				deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
					AlwaysZeroGasContracts: []string{targetAddr.Hex()},
				})
			},
			txSetup: func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
				targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
				tx := evm.NewTx(&evm.EvmTxArgs{
					ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
					Nonce:    0,
					GasLimit: 50_000,
					GasPrice: big.NewInt(0),
					To:       &targetAddr,
					Amount:   big.NewInt(0),
				})
				tx.From = deps.Sender.EthAddr.Hex()
				return tx
			},
			wantErr:         "",
			zeroGasEligible: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			sdb := deps.NewStateDB()

			tc.beforeTxSetup(&deps, sdb)
			tx := tc.txSetup(&deps)

			deps.SetCtx(deps.Ctx().WithIsCheckTx(true))
			simulate := false
			unusedOpts := AnteOptionsForTests{MaxTxGasWanted: 0}

			if tc.zeroGasEligible {
				err := evmante.AnteStepDetectZeroGas(sdb, sdb.Keeper(), tx, simulate, unusedOpts)
				s.Require().NoError(err)
				s.Require().True(evm.IsZeroGasEthTx(sdb.Ctx()))
			}

			err := evmante.AnteStepVerifyEthAcc(
				sdb, sdb.Keeper(), tx, simulate, unusedOpts,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			if tc.zeroGasEligible {
				acc := sdb.Keeper().GetAccount(sdb.Ctx(), tx.FromAddr())
				s.Require().NotNil(acc, "sender account should exist after VerifyEthAcc for zero-gas tx")
			}
		})
	}
}

func happyGasLimit() *big.Int {
	return new(big.Int).SetUint64(
		gethparams.TxGasContractCreation + 888,
		// 888 is a cushion to account for KV store reads and writes
	)
}

// TestCanTransfer_ZeroGas_RunsAndPasses documents that CanTransfer runs (does not skip)
// for zero-gas txs and passes. EffectiveGasFeeCapWei returns max(baseFee, txCap), so the
// gas cap check passes; value is 0 by eligibility so the value check no-ops.
func TestCanTransfer_ZeroGas_RunsAndPasses(t *testing.T) {
	targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	deps := evmtest.NewTestDeps()

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

	gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
	err := tx.Sign(gethSigner, deps.Sender.KeyringSigner)
	require.NoError(t, err)

	err = evmante.AnteStepDetectZeroGas(sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED)
	require.NoError(t, err)
	require.True(t, evm.IsZeroGasEthTx(sdb.Ctx()))

	err = evmante.AnteStepCanTransfer(sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED)
	require.NoError(t, err, "CanTransfer must run and pass for zero-gas tx")
}

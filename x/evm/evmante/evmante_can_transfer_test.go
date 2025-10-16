package evmante_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
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
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *evmstate.SDB)
		txSetup       func(deps *evmtest.TestDeps) *evm.MsgEthereumTx
		wantErr       string
	}{
		{
			name: "happy: sender with funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
				AddBalanceSigned(sdb, deps.Sender.EthAddr, evm.NativeToWei(happyGasLimit()))
			},
			txSetup: evmtest.HappyCreateContractTx,
			wantErr: "",
		},
		{
			name:          "sad: sender has insufficient gas balance",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {},
			txSetup:       evmtest.HappyCreateContractTx,
			wantErr:       "sender balance < tx cost",
		},
		{
			name:          "sad: invalid tx",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {},
			txSetup: func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
				return new(evm.MsgEthereumTx)
			},
			wantErr: "failed to unpack tx data",
		},
		{
			name:          "sad: empty from addr",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {},
			txSetup: func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps)
				tx.From = ""
				return tx
			},
			wantErr: "from address cannot be empty",
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
			err := evmante.AnteStepVerifyEthAcc(
				sdb, sdb.Keeper(), tx, simulate, unusedOpts,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

func happyGasLimit() *big.Int {
	return new(big.Int).SetUint64(
		gethparams.TxGasContractCreation + 888,
		// 888 is a cushion to account for KV store reads and writes
	)
}

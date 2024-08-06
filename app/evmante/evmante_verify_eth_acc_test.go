package evmante_test

import (
	"math/big"

	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/app/evmante"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

func (s *TestSuite) TestAnteDecoratorVerifyEthAcc_CheckTx() {
	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB)
		txSetup       func(deps *evmtest.TestDeps) *evm.MsgEthereumTx
		wantErr       string
	}{
		{
			name: "happy: sender with funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				sdb.AddBalance(deps.Sender.EthAddr, evm.NativeToWei(happyGasLimit()))
			},
			txSetup: evmtest.HappyCreateContractTx,
			wantErr: "",
		},
		{
			name:          "sad: sender has insufficient gas balance",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {},
			txSetup:       evmtest.HappyCreateContractTx,
			wantErr:       "sender balance < tx cost",
		},
		{
			name: "sad: sender cannot be a contract -> no contract bytecode",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				// Force account to be a smart contract
				sdb.SetCode(deps.Sender.EthAddr, []byte("evm bytecode stuff"))
			},
			txSetup: evmtest.HappyCreateContractTx,
			wantErr: "sender is not EOA",
		},
		{
			name:          "sad: invalid tx",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {},
			txSetup: func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
				return new(evm.MsgEthereumTx)
			},
			wantErr: "failed to unpack tx data",
		},
		{
			name:          "sad: empty from addr",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {},
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
			stateDB := deps.StateDB()
			anteDec := evmante.NewAnteDecVerifyEthAcc(&deps.Chain.AppKeepers.EvmKeeper, &deps.Chain.AppKeepers.AccountKeeper)

			tc.beforeTxSetup(&deps, stateDB)
			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			deps.Ctx = deps.Ctx.WithIsCheckTx(true)
			_, err := anteDec.AnteHandle(
				deps.Ctx, tx, false, evmtest.NextNoOpAnteHandler,
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

package evmante_test

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

func (s *TestSuite) TestAnteDecoratorVerifyEthAcc_CheckTx() {
	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB, wnibi gethcommon.Address)
		txSetup       func(deps *evmtest.TestDeps) *evm.MsgEthereumTx
		wantErr       string
	}{
		{
			name: "happy: sender with funds (native)",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {
				sdb.AddBalanceSigned(deps.Sender.EthAddr, evm.NativeToWei(happyGasLimit()))
			},
			txSetup: evmtest.HappyCreateContractTx,
			wantErr: "",
		},
		{
			name: "happy: sender with funds (wnibi)",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, wnibi gethcommon.Address) {
				acc := deps.App.AccountKeeper.NewAccountWithAddress(deps.Ctx, eth.EthAddrToNibiruAddr(deps.Sender.EthAddr))
				deps.App.AccountKeeper.SetAccount(deps.Ctx, acc)
				// Fund the contract wnibi with unibi and set the mapping slot of the sender to be the equivalent
				// of happyGasLimit in unibi, so that the sender has enough wnibi to pay for gas
				sdb.AddBalanceSigned(wnibi, evm.NativeToWei(happyGasLimit()))
				slot := CalcMappingSlot(deps.Sender.EthAddr, 3)
				value := gethcommon.BigToHash(evm.NativeToWei(happyGasLimit()))
				sdb.SetState(wnibi, slot, value)
			},
			txSetup: evmtest.HappyCreateContractTx,
			wantErr: "",
		},
		{
			name:          "sad: sender has insufficient gas balance",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {},
			txSetup:       evmtest.HappyCreateContractTx,
			wantErr:       "sender balance < tx cost",
		},
		{
			name: "sad: sender cannot be a contract -> no contract bytecode",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {
				// Force account to be a smart contract
				sdb.SetCode(deps.Sender.EthAddr, []byte("evm bytecode stuff"))
			},
			txSetup: evmtest.HappyCreateContractTx,
			wantErr: "sender is not EOA",
		},
		{
			name:          "sad: invalid tx",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {},
			txSetup: func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
				return new(evm.MsgEthereumTx)
			},
			wantErr: "failed to unpack tx data",
		},
		{
			name:          "sad: empty from addr",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {},
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
			deps.DeployWNIBI(&s.Suite)
			stateDB := deps.NewStateDB()
			anteDec := evmante.NewAnteDecVerifyEthAcc(deps.App.EvmKeeper, &deps.App.AccountKeeper)

			tc.beforeTxSetup(&deps, stateDB, deps.App.EvmKeeper.GetParams(deps.Ctx).CanonicalWnibi.Address)
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

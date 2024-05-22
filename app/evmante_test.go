package app_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"

	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

var NextNoOpAnteHandler sdk.AnteHandler = func(
	ctx sdk.Context, tx sdk.Tx, simulate bool,
) (newCtx sdk.Context, err error) {
	return ctx, nil
}

func (s *TestSuite) TestAnteDecoratorVerifyEthAcc_CheckTx() {
	happyCreateContractTx := func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
		ethContractCreationTxParams := &evm.EvmTxArgs{
			ChainID:  deps.Chain.EvmKeeper.EthChainID(deps.Ctx),
			Nonce:    1,
			Amount:   big.NewInt(10),
			GasLimit: 1000,
			GasPrice: big.NewInt(1),
		}
		tx := evm.NewTx(ethContractCreationTxParams)
		tx.From = deps.Sender.EthAddr.Hex()
		return tx
	}

	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB)
		txSetup       func(deps *evmtest.TestDeps) *evm.MsgEthereumTx
		wantErr       string
	}{
		{
			name: "happy: sender with funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				gasLimit := new(big.Int).SetUint64(
					gethparams.TxGasContractCreation + 500,
				)
				// Force account to be a smart contract
				sdb.AddBalance(deps.Sender.EthAddr, gasLimit)
			},
			txSetup: happyCreateContractTx,
			wantErr: "",
		},
		{
			name:          "sad: sender has insufficient gas balance",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {},
			txSetup:       happyCreateContractTx,
			wantErr:       "sender balance < tx cost",
		},
		{
			name: "sad: sender cannot be a contract -> no contract bytecode",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				// Force account to be a smart contract
				sdb.SetCode(deps.Sender.EthAddr, []byte("evm bytecode stuff"))
			},
			txSetup: happyCreateContractTx,
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
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.StateDB()
			anteDec := app.NewAnteDecVerifyEthAcc(deps.Chain.AppKeepers)

			tc.beforeTxSetup(&deps, stateDB)
			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			deps.Ctx = deps.Ctx.WithIsCheckTx(true)
			_, err := anteDec.AnteHandle(
				deps.Ctx, tx, false, NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

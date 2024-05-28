package app_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth"
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
	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB)
		txSetup       func(deps *evmtest.TestDeps) *evm.MsgEthereumTx
		wantErr       string
	}{
		{
			name: "happy: sender with funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				sdb.AddBalance(deps.Sender.EthAddr, happyGasLimit())
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
		{
			name:          "sad: empty from addr",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {},
			txSetup: func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
				tx := happyCreateContractTx(deps)
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

func happyGasLimit() *big.Int {
	return new(big.Int).SetUint64(
		gethparams.TxGasContractCreation + 888,
		// 888 is a cushion to account for KV store reads and writes
	)
}

func gasLimitCreateContract() *big.Int {
	return new(big.Int).SetUint64(
		gethparams.TxGasContractCreation + 700,
	)
}

func happyCreateContractTx(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
	ethContractCreationTxParams := &evm.EvmTxArgs{
		ChainID:  deps.Chain.EvmKeeper.EthChainID(deps.Ctx),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: gasLimitCreateContract().Uint64(),
		GasPrice: big.NewInt(1),
	}
	tx := evm.NewTx(ethContractCreationTxParams)
	tx.From = deps.Sender.EthAddr.Hex()
	return tx
}

func (s *TestSuite) TestAnteDecEthGasConsume() {
	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB)
		txSetup       func(deps *evmtest.TestDeps) *evm.MsgEthereumTx
		wantErr       string
		maxGasWanted  uint64
		gasMeter      sdk.GasMeter
	}{
		{
			name: "happy: sender with funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				gasLimit := happyGasLimit()
				balance := new(big.Int).Add(gasLimit, big.NewInt(100))
				sdb.AddBalance(deps.Sender.EthAddr, balance)
			},
			txSetup:      happyCreateContractTx,
			wantErr:      "",
			gasMeter:     eth.NewInfiniteGasMeterWithLimit(happyGasLimit().Uint64()),
			maxGasWanted: 0,
		},
		{
			name: "happy: is recheck tx",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				deps.Ctx = deps.Ctx.WithIsReCheckTx(true)
			},
			txSetup:  happyCreateContractTx,
			gasMeter: eth.NewInfiniteGasMeterWithLimit(0),
			wantErr:  "",
		},
		{
			name: "sad: out of gas",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				gasLimit := happyGasLimit()
				balance := new(big.Int).Add(gasLimit, big.NewInt(100))
				sdb.AddBalance(deps.Sender.EthAddr, balance)
			},
			txSetup:      happyCreateContractTx,
			wantErr:      "exceeds block gas limit (0)",
			gasMeter:     eth.NewInfiniteGasMeterWithLimit(0),
			maxGasWanted: 0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.StateDB()
			anteDec := app.NewAnteDecEthGasConsume(
				deps.Chain.AppKeepers, tc.maxGasWanted,
			)

			tc.beforeTxSetup(&deps, stateDB)
			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			deps.Ctx = deps.Ctx.WithIsCheckTx(true)
			deps.Ctx = deps.Ctx.WithBlockGasMeter(tc.gasMeter)
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

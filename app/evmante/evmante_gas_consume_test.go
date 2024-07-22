package evmante_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app/evmante"
	"github.com/NibiruChain/nibiru/eth"
	evmtestutil "github.com/NibiruChain/nibiru/x/common/testutil/evm"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

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
			txSetup:      evmtestutil.HappyCreateContractTx,
			wantErr:      "",
			gasMeter:     eth.NewInfiniteGasMeterWithLimit(happyGasLimit().Uint64()),
			maxGasWanted: 0,
		},
		{
			name: "happy: is recheck tx",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				deps.Ctx = deps.Ctx.WithIsReCheckTx(true)
			},
			txSetup:  evmtestutil.HappyCreateContractTx,
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
			txSetup:      evmtestutil.HappyCreateContractTx,
			wantErr:      "exceeds block gas limit (0)",
			gasMeter:     eth.NewInfiniteGasMeterWithLimit(0),
			maxGasWanted: 0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.StateDB()
			anteDec := evmante.NewAnteDecEthGasConsume(
				&deps.Chain.AppKeepers.EvmKeeper, tc.maxGasWanted,
			)

			tc.beforeTxSetup(&deps, stateDB)
			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			deps.Ctx = deps.Ctx.WithIsCheckTx(true)
			deps.Ctx = deps.Ctx.WithBlockGasMeter(tc.gasMeter)
			_, err := anteDec.AnteHandle(
				deps.Ctx, tx, false, evmtestutil.NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

package app_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
	"github.com/NibiruChain/nibiru/x/evm/types"
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
		txSetup       func(deps *evmtest.TestDeps) *types.MsgEthereumTx
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
			txSetup: func(deps *evmtest.TestDeps) *types.MsgEthereumTx {
				return new(types.MsgEthereumTx)
			},
			wantErr: "failed to unpack tx data",
		},
		{
			name:          "sad: empty from addr",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {},
			txSetup: func(deps *evmtest.TestDeps) *types.MsgEthereumTx {
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

func happyCreateContractTx(deps *evmtest.TestDeps) *types.MsgEthereumTx {
	ethContractCreationTxParams := &types.EvmTxArgs{
		ChainID:  deps.Chain.EvmKeeper.EthChainID(deps.Ctx),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: gasLimitCreateContract().Uint64(),
		GasPrice: big.NewInt(1),
	}
	tx := types.NewTx(ethContractCreationTxParams)
	tx.From = deps.Sender.EthAddr.Hex()
	return tx
}

func happyTransfertTx(deps *evmtest.TestDeps, nonce uint64) *types.MsgEthereumTx {
	to := evmtest.NewEthAccInfo().EthAddr
	ethContractCreationTxParams := &types.EvmTxArgs{
		ChainID:  deps.Chain.EvmKeeper.EthChainID(deps.Ctx),
		Nonce:    nonce,
		Amount:   big.NewInt(10),
		GasLimit: gasLimitCreateContract().Uint64(),
		GasPrice: big.NewInt(1),
		To:       &to,
	}
	tx := types.NewTx(ethContractCreationTxParams)
	tx.From = deps.Sender.EthAddr.Hex()
	return tx
}

func nonEvmMsgTx(deps *evmtest.TestDeps) sdk.Tx {
	gasLimit := uint64(10)
	fees := sdk.NewCoins(sdk.NewInt64Coin("unibi", int64(gasLimit)))
	msg := &banktypes.MsgSend{
		FromAddress: deps.Sender.NibiruAddr.String(),
		ToAddress:   evmtest.NewEthAccInfo().NibiruAddr.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("unibi", 1)),
	}
	return buildTx(deps, true, msg, gasLimit, fees)
}

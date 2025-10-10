package evmante_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *TestSuite) TestCanTransfer() {
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

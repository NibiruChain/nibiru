package evmante_test

import (
	"math/big"

	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

var InvalidChainID = big.NewInt(987654321)

func (s *TestSuite) TestEthSigVerificationDecorator() {
	testCases := []struct {
		name    string
		txSetup func(deps *evmtest.TestDeps) evm.Tx
		wantErr string
	}{
		{
			name: "happy: signed ethereum tx",
			txSetup: func(deps *evmtest.TestDeps) evm.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
				err := tx.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "",
		},
		{
			name: "sad: unsigned tx",
			txSetup: func(deps *evmtest.TestDeps) evm.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				return tx
			},
			wantErr: "couldn't retrieve sender address from the ethereum transaction: invalid transaction v, r, s values: tx intended signer does not match the given signer",
		},
		{
			name: "sad: ethereum tx invalid chain id",
			txSetup: func(deps *evmtest.TestDeps) evm.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				gethSigner := gethcore.LatestSignerForChainID(InvalidChainID)
				err := tx.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "invalid chain id for signer",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			sdb := deps.NewStateDB()
			deps.SetCtx(deps.Ctx().WithIsCheckTx(true))

			tx := tc.txSetup(&deps)

			simulate := false

			err := evmante.EthSigVerification(
				sdb,
				sdb.Keeper(),
				tx,
				simulate,
				ANTE_OPTIONS_UNUSED,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

var ANTE_OPTIONS_UNUSED = AnteOptionsForTests{MaxTxGasWanted: 0}

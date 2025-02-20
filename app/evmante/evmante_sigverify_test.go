package evmante_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	tf "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

var InvalidChainID = big.NewInt(987654321)

func (s *TestSuite) TestEthSigVerificationDecorator() {
	testCases := []struct {
		name    string
		txSetup func(deps *evmtest.TestDeps) sdk.Tx
		wantErr string
	}{
		{
			name: "sad: unsigned tx",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				return tx
			},
			wantErr: "rejected unprotected Ethereum transaction",
		},
		{
			name: "sad: non ethereum tx",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return legacytx.StdTx{
					Msgs: []sdk.Msg{
						&tf.MsgMint{},
					},
				}
			},
			wantErr: "invalid message",
		},
		{
			name: "sad: ethereum tx invalid chain id",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				gethSigner := gethcore.LatestSignerForChainID(InvalidChainID)
				err := tx.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "invalid chain id for signer",
		},
		{
			name: "happy: signed ethereum tx",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx))
				err := tx.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.NewStateDB()
			anteDec := evmante.NewEthSigVerificationDecorator(deps.App.AppKeepers.EvmKeeper)

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

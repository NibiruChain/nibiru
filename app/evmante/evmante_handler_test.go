package evmante_test

import (
	"math/big"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

func (s *TestSuite) TestAnteHandlerEVM() {
	testCases := []struct {
		name          string
		txSetup       func(deps *evmtest.TestDeps) sdk.FeeTx
		ctxSetup      func(deps *evmtest.TestDeps)
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB)
		wantErr       string
	}{
		{
			name: "happy: signed tx, sufficient funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				balanceMicronibi := new(big.Int).Add(evmtest.GasLimitCreateContract(), big.NewInt(100))
				sdb.AddBalance(
					deps.Sender.EthAddr,
					evm.NativeToWei(balanceMicronibi),
				)
			},
			ctxSetup: func(deps *evmtest.TestDeps) {
				gasPrice := sdk.NewInt64Coin("unibi", 1)
				maxGasMicronibi := new(big.Int).Add(evmtest.GasLimitCreateContract(), big.NewInt(100))
				cp := &tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{
						MaxGas: evm.NativeToWei(maxGasMicronibi).Int64(),
					},
				}
				deps.Ctx = deps.Ctx.
					WithMinGasPrices(
						sdk.NewDecCoins(sdk.NewDecCoinFromCoin(gasPrice)),
					).
					WithIsCheckTx(true).
					WithConsensusParams(cp)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				txMsg := evmtest.HappyTransferTx(deps, 0)
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()

				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx))
				err := txMsg.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)

				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)

				return tx
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.StateDB()

			anteHandlerEVM := evmante.NewAnteHandlerEVM(
				ante.AnteHandlerOptions{
					HandlerOptions: authante.HandlerOptions{
						AccountKeeper:          deps.App.AccountKeeper,
						BankKeeper:             deps.App.BankKeeper,
						FeegrantKeeper:         deps.App.FeeGrantKeeper,
						SignModeHandler:        deps.EncCfg.TxConfig.SignModeHandler(),
						SigGasConsumer:         authante.DefaultSigVerificationGasConsumer,
						ExtensionOptionChecker: func(*codectypes.Any) bool { return true },
					},
					EvmKeeper:     deps.App.EvmKeeper,
					AccountKeeper: deps.App.AccountKeeper,
				})

			tx := tc.txSetup(&deps)

			if tc.ctxSetup != nil {
				tc.ctxSetup(&deps)
			}
			if tc.beforeTxSetup != nil {
				tc.beforeTxSetup(&deps, stateDB)
				err := stateDB.Commit()
				s.Require().NoError(err)
			}

			_, err := anteHandlerEVM(
				deps.Ctx, tx, false,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

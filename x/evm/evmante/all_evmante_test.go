package evmante_test

import (
	"math/big"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func (s *Suite) TestAnteHandlerEVM() {
	testCases := []struct {
		name          string
		txSetup       func(deps *evmtest.TestDeps) sdk.FeeTx
		ctxSetup      func(deps *evmtest.TestDeps)
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *evmstate.SDB)
		wantErr       string
		onSuccess     func(newCtx sdk.Context)
	}{
		{
			name: "happy: signed tx, sufficient funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
				balanceMicronibi := new(big.Int).Add(evmtest.GasLimitCreateContract(), big.NewInt(100))
				AddBalanceSigned(sdb,
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
				deps.SetCtx(deps.Ctx().
					WithMinGasPrices(
						sdk.NewDecCoins(sdk.NewDecCoinFromCoin(gasPrice)),
					).
					WithIsCheckTx(true).
					WithConsensusParams(cp),
				)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				txMsg := evmtest.HappyTransferTx(deps, 0)
				txBuilder := deps.App.GetTxConfig().NewTxBuilder()

				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
				err := txMsg.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)

				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)

				return tx
			},
			wantErr: "",
			onSuccess: func(newCtx sdk.Context) {
				s.Require().False(evm.IsZeroGasEthTx(newCtx), "expected IsZeroGasEthTx to be false for normal tx")
			},
		},
		{
			name: "zero-gas: allowlisted contract, meta populated after all ante steps",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
				targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
				deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
					AlwaysZeroGasContracts: []string{targetAddr.Hex()},
				})
				balanceMicronibi := new(big.Int).Add(evmtest.GasLimitCreateContract(), big.NewInt(100))
				AddBalanceSigned(sdb, deps.Sender.EthAddr, evm.NativeToWei(balanceMicronibi))
			},
			ctxSetup: func(deps *evmtest.TestDeps) {
				gasPrice := sdk.NewInt64Coin("unibi", 1)
				maxGasMicronibi := new(big.Int).Add(evmtest.GasLimitCreateContract(), big.NewInt(100))
				cp := &tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{
						MaxGas: evm.NativeToWei(maxGasMicronibi).Int64(),
					},
				}
				deps.SetCtx(deps.Ctx().
					WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoinFromCoin(gasPrice))).
					WithIsCheckTx(true).
					WithConsensusParams(cp),
				)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
				txMsg := evm.NewTx(&evm.EvmTxArgs{
					ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
					Nonce:    0,
					GasLimit: 50_000,
					GasPrice: big.NewInt(1),
					To:       &targetAddr,
					Amount:   big.NewInt(0),
				})
				txMsg.From = deps.Sender.EthAddr.Hex()
				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
				err := txMsg.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)
				txBuilder := deps.App.GetTxConfig().NewTxBuilder()
				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "",
			onSuccess: func(newCtx sdk.Context) {
				s.Require().True(evm.IsZeroGasEthTx(newCtx), "expected IsZeroGasEthTx to be true for zero-gas tx")
			},
		},
		{
			name: "zero-gas: sender with no balance, full ante passes (first-time onboarding)",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
				targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
				deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
					AlwaysZeroGasContracts: []string{targetAddr.Hex()},
				})
				// No AddBalanceSigned: sender has no gas balance; VerifyEthAcc still runs account creation and skips balance check.
			},
			ctxSetup: func(deps *evmtest.TestDeps) {
				gasPrice := sdk.NewInt64Coin("unibi", 1)
				maxGasMicronibi := new(big.Int).Add(evmtest.GasLimitCreateContract(), big.NewInt(100))
				cp := &tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{
						MaxGas: evm.NativeToWei(maxGasMicronibi).Int64(),
					},
				}
				deps.SetCtx(deps.Ctx().
					WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoinFromCoin(gasPrice))).
					WithIsCheckTx(true).
					WithConsensusParams(cp),
				)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
				txMsg := evm.NewTx(&evm.EvmTxArgs{
					ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
					Nonce:    0,
					GasLimit: 50_000,
					GasPrice: big.NewInt(0),
					To:       &targetAddr,
					Amount:   big.NewInt(0),
				})
				txMsg.From = deps.Sender.EthAddr.Hex()
				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
				err := txMsg.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)
				txBuilder := deps.App.GetTxConfig().NewTxBuilder()
				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "",
			onSuccess: func(newCtx sdk.Context) {
				s.Require().True(evm.IsZeroGasEthTx(newCtx), "expected IsZeroGasEthTx to be true for zero-gas tx")
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			sdb := deps.NewStateDB()

			anteHandlerEVM := evmante.NewAnteHandlerEvm(
				ante.AnteHandlerOptions{
					HandlerOptions: authante.HandlerOptions{
						AccountKeeper:          deps.App.AccountKeeper,
						BankKeeper:             deps.App.BankKeeper,
						FeegrantKeeper:         deps.App.FeeGrantKeeper,
						SignModeHandler:        deps.App.GetTxConfig().SignModeHandler(),
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
				tc.beforeTxSetup(&deps, sdb)
				sdb.Commit()
			}

			newCtx, err := anteHandlerEVM(
				deps.Ctx(), tx, false,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			if tc.onSuccess != nil {
				tc.onSuccess(newCtx)
			}
		})
	}
}

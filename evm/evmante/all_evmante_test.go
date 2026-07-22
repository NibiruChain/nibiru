package evmante_test

import (
	"math/big"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authante "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/ante"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
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
		{
			name: "zero-gas: nonzero value with exact value balance passes full ante",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
				targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
				deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
					AlwaysZeroGasContracts: []string{targetAddr.Hex()},
				})
				AddBalanceSigned(sdb, deps.Sender.EthAddr, big.NewInt(123))
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
					GasPrice: evm.NativeToWei(big.NewInt(1)),
					To:       &targetAddr,
					Amount:   big.NewInt(123),
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
				s.Require().True(evm.IsZeroGasEthTx(newCtx), "expected IsZeroGasEthTx to be true for payable zero-gas tx")
			},
		},
		{
			name: "zero-gas: nonzero value with insufficient value balance fails full ante",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
				targetAddr := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
				deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
					AlwaysZeroGasContracts: []string{targetAddr.Hex()},
				})
				AddBalanceSigned(sdb, deps.Sender.EthAddr, big.NewInt(122))
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
					Amount:   big.NewInt(123),
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
			wantErr: "failed to transfer 123 wei",
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

func (s *Suite) TestAnteHandlerEVMCheckTxNonceSequence() {
	deps := evmtest.NewTestDeps()
	sdb := deps.NewStateDB()
	maxGasMicronibi := new(big.Int).Add(evmtest.GasLimitCreateContract(), big.NewInt(100))
	AddBalanceSigned(sdb, deps.Sender.EthAddr, evm.NativeToWei(maxGasMicronibi))
	sdb.Commit()
	deps.App.Commit()

	makeTxBytes := func(nonce uint64) []byte {
		txMsg := evmtest.HappyTransferTx(&deps, nonce)
		gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
		s.Require().NoError(txMsg.Sign(gethSigner, deps.Sender.KeyringSigner))

		txBuilder := deps.App.GetTxConfig().NewTxBuilder()
		tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
		s.Require().NoError(err)
		txBytes, err := deps.App.GetTxConfig().TxEncoder()(tx)
		s.Require().NoError(err)
		return txBytes
	}
	checkTx := func(txBytes []byte, checkType abci.CheckTxType) abci.ResponseCheckTx {
		return deps.App.CheckTx(abci.RequestCheckTx{
			Tx:   txBytes,
			Type: checkType,
		})
	}
	liveTxs := make(map[uint64][]byte)
	admit := func(nonce uint64) abci.ResponseCheckTx {
		txBytes := makeTxBytes(nonce)
		resp := checkTx(txBytes, abci.CheckTxType_New)
		if resp.IsOK() {
			liveTxs[nonce] = txBytes
		}
		return resp
	}
	recheck := func(nonce uint64) abci.ResponseCheckTx {
		return checkTx(liveTxs[nonce], abci.CheckTxType_Recheck)
	}

	resp := admit(1_000)
	s.Require().False(resp.IsOK())
	s.Require().Contains(resp.Log, "future nonce gap too large")

	// A future nonce is admitted initially but purged when the state nonce chain
	// is incomplete.
	s.Require().True(admit(2).IsOK())
	resp = recheck(2)
	s.Require().False(resp.IsOK())
	s.Require().Contains(resp.Log, "state nonce chain is incomplete")
	delete(liveTxs, 2)

	s.Require().True(admit(0).IsOK())
	s.Require().True(admit(1).IsOK())
	s.Require().True(admit(2).IsOK())
	s.Require().True(recheck(2).IsOK(), "complete state nonce chain must survive recheck")

	for nonce := uint64(3); nonce < evmante.MaxPendingTxsPerSender; nonce++ {
		s.Require().True(admit(nonce).IsOK())
	}

	resp = admit(evmante.MaxPendingTxsPerSender)
	s.Require().False(resp.IsOK())
	s.Require().Contains(resp.Log, "sender slot limit reached")

	resp = checkTx(makeTxBytes(1), abci.CheckTxType_New)
	s.Require().False(resp.IsOK())
	s.Require().Contains(resp.Log, "nonce slot already occupied")

	blockHeader := deps.Ctx().BlockHeader()
	blockHeader.Height++
	deps.App.BeginBlock(abci.RequestBeginBlock{Header: blockHeader})
	deps.App.EndBlock(abci.RequestEndBlock{Height: blockHeader.Height})
	deps.App.Commit()

	// EndBlock does not reset live slots. They remain until block inclusion or a
	// failed CheckTxType_Recheck removes them.
	s.Require().Equal(int(evmante.MaxPendingTxsPerSender), deps.App.EvmMempool.CountTx())
	resp = checkTx(makeTxBytes(0), abci.CheckTxType_New)
	s.Require().False(resp.IsOK())
	s.Require().Contains(resp.Log, "nonce slot already occupied")
}

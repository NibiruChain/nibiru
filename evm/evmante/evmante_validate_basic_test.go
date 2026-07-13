package evmante_test

import (
	"math/big"

	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authtx "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/tx"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func (s *Suite) TestAnteStepValidateBasic() {
	evmAnteStep := evmante.AnteStepValidateBasic
	priorSteps := []evmante.AnteStep{
		evmante.EthSigVerification,
	}
	testCases := []AnteTC{
		{
			Name:        "happy: valid ethereum transaction after signature verification",
			PriorSteps:  priorSteps,
			EvmAnteStep: evmAnteStep,
			TxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx {
				// Create a properly signed transaction like in EthSigVerification tests
				tx := evmtest.HappyCreateContractTx(deps)
				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
				err := tx.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err, "Failed to sign transaction")

				// Run EthSigVerification to set the From field
				err = evmante.EthSigVerification(
					sdb, deps.App.EvmKeeper, tx, false, ANTE_OPTIONS_UNUSED)
				s.Require().NoError(err, "EthSigVerification failed")
				return tx
			},
			WantErr: "",
		},
		{
			Name:        "sad: unsigned tx",
			PriorSteps:  priorSteps,
			EvmAnteStep: evmAnteStep,
			TxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				return tx
			},
			WantPriorStepErr: "couldn't retrieve sender address from the ethereum transaction: invalid transaction v, r, s values: tx intended signer does not match the given signer",
		},
		{
			Name:        "happy: ReCheckTx skips validation",
			EvmAnteStep: evmAnteStep,
			TxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				sdb.SetCtx(sdb.Ctx().WithIsReCheckTx(true))
				return tx
			},
			WantErr: "",
		},
		{
			Name:        "sad: invalid chain id in prior step (EthSigVerification)",
			PriorSteps:  priorSteps,
			EvmAnteStep: evmAnteStep,
			TxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				invalidSigner := gethcore.LatestSignerForChainID(InvalidChainID)
				err := tx.Sign(invalidSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)
				return tx
			},
			WantPriorStepErr: "invalid chain id for signer",
		},
		{
			Name:        "sad: gas limit below intrinsic cost",
			PriorSteps:  priorSteps,
			EvmAnteStep: evmAnteStep,
			TxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx {
				lowGas := gethparams.TxGas - 1
				args := &evm.EvmTxArgs{
					ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx()),
					Nonce:    0,
					Amount:   big.NewInt(1),
					GasLimit: lowGas,
					GasPrice: big.NewInt(1),
					To:       nil,
				}
				msgEthTx := evm.NewTx(args)
				signer := gethcore.LatestSignerForChainID(
					deps.App.EvmKeeper.EthChainID(deps.Ctx()),
				)
				msgEthTx.From = deps.Sender.EthAddr.Hex()
				err := msgEthTx.Sign(signer, deps.Sender.KeyringSigner)
				s.Require().NoError(err)
				return msgEthTx
			},
			WantErr: "tx gas limit is less than intrinsic gas cost",
		},
	}

	RunAnteTCs(&s.Suite, testCases)
}

func (s *Suite) TestAnteStepValidateBasic_ZeroGasSkipsDynamicFeeOnlyValidation() {
	deps := evmtest.NewTestDeps()
	sdb := deps.NewStateDB()
	to := evmtest.NewEthPrivAcc().EthAddr
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{to.Hex()},
	})

	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:   deps.App.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:     0,
		GasLimit:  50_000,
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(2),
		Amount:    big.NewInt(0),
		To:        &to,
	})
	tx.From = deps.Sender.EthAddr.Hex()
	s.Require().NoError(tx.Sign(
		gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx())),
		deps.Sender.KeyringSigner,
	))
	s.Require().ErrorContains(tx.ValidateBasic(), "max priority fee per gas higher than max fee per gas")

	s.Require().NoError(evmante.EthSigVerification(sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED))
	s.Require().NoError(evmante.AnteStepDetectZeroGas(sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED))
	s.Require().True(evm.IsZeroGasEthTx(sdb.Ctx()))
	s.Require().NoError(evmante.AnteStepValidateBasic(sdb, sdb.Keeper(), tx, false, ANTE_OPTIONS_UNUSED))
}

// buildTx constructs a Cosmos SDK tx (optionally with Ethereum extension options)
// from a given sdk.Msg, gasLimit and fees, using the test deps' tx config.
func buildTx(
	deps *evmtest.TestDeps,
	ethExtentions bool,
	msg sdk.Msg,
	gasLimit uint64,
	fees sdk.Coins,
) sdk.FeeTx {
	txBuilder, _ := deps.App.GetTxConfig().NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
	if ethExtentions {
		option, _ := codectypes.NewAnyWithValue(&evm.ExtensionOptionsEthereumTx{})
		txBuilder.SetExtensionOptions(option)
	}
	err := txBuilder.SetMsgs(msg)
	if err != nil {
		panic(err)
	}
	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetFeeAmount(fees)

	return txBuilder.GetTx()
}

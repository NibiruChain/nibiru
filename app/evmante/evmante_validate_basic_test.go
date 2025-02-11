package evmante_test

import (
	"math/big"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *TestSuite) TestEthValidateBasicDecorator() {
	testCases := []struct {
		name        string
		ctxSetup    func(deps *evmtest.TestDeps)
		txSetup     func(deps *evmtest.TestDeps) sdk.Tx
		paramsSetup func(deps *evmtest.TestDeps) evm.Params
		wantErr     string
	}{
		{
			name: "happy: properly built eth tx",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
				tx, err := evmtest.HappyCreateContractTx(deps).BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "",
		},
		{
			name: "sad: fail to set params",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return evmtest.HappyCreateContractTx(deps)
			},
			paramsSetup: func(deps *evmtest.TestDeps) evm.Params {
				return evm.Params{
					CreateFuntokenFee: sdk.NewInt(-1),
				}
			},
			wantErr: "createFuntokenFee cannot be negative: -1",
		},
		{
			name: "happy: ctx recheck should ignore validation",
			ctxSetup: func(deps *evmtest.TestDeps) {
				deps.Ctx = deps.Ctx.WithIsReCheckTx(true)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return evmtest.HappyCreateContractTx(deps)
			},
			wantErr: "",
		},
		{
			name: "sad: fail chain id basic validation",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return evmtest.HappyCreateContractTx(deps)
			},
			wantErr: "invalid chain-id",
		},
		{
			name: "sad: tx not implementing protoTxProvider",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				gethSigner := gethcore.LatestSignerForChainID(InvalidChainID)
				err := tx.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "didn't implement interface protoTxProvider",
		},
		{
			name: "sad: eth tx with memo should fail",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
				txBuilder.SetMemo("memo")
				tx, err := evmtest.HappyCreateContractTx(deps).BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "invalid request",
		},
		{
			name: "sad: eth tx with fee payer should fail",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
				txBuilder.SetFeePayer(testutil.AccAddress())
				tx, err := evmtest.HappyCreateContractTx(deps).BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "invalid request",
		},
		{
			name: "sad: eth tx with fee granter should fail",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
				txBuilder.SetFeeGranter(testutil.AccAddress())
				tx, err := evmtest.HappyCreateContractTx(deps).BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "invalid request",
		},
		{
			name: "sad: eth tx with signatures should fail",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
				sigV2 := signing.SignatureV2{
					PubKey: deps.Sender.PrivKey.PubKey(),
					Data: &signing.SingleSignatureData{
						SignMode:  deps.EncCfg.TxConfig.SignModeHandler().DefaultMode(),
						Signature: nil,
					},
					Sequence: 0,
				}
				err := txBuilder.SetSignatures(sigV2)
				s.Require().NoError(err)
				txMsg := evmtest.HappyCreateContractTx(deps)

				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx))
				err = txMsg.Sign(gethSigner, deps.Sender.KeyringSigner)
				s.Require().NoError(err)

				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "tx AuthInfo SignerInfos should be empty",
		},
		{
			name: "sad: tx without extension options should fail",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				chainID := deps.App.EvmKeeper.EthChainID(deps.Ctx)
				gasLimit := uint64(10)
				fees := sdk.NewCoins(sdk.NewInt64Coin("unibi", int64(gasLimit)))
				msg := buildEthMsg(chainID, gasLimit, deps.Sender.NibiruAddr.String(), nil)
				return buildTx(deps, false, msg, gasLimit, fees)
			},
			wantErr: "for eth tx length of ExtensionOptions should be 1",
		},
		{
			name: "sad: tx with non evm message",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				gasLimit := uint64(10)
				fees := sdk.NewCoins(sdk.NewInt64Coin("unibi", int64(gasLimit)))
				msg := &banktypes.MsgSend{
					FromAddress: deps.Sender.NibiruAddr.String(),
					ToAddress:   evmtest.NewEthPrivAcc().NibiruAddr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin("unibi", 1)),
				}
				return buildTx(deps, true, msg, gasLimit, fees)
			},
			wantErr: "invalid message",
		},
		{
			name: "sad: tx with from value set should fail",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				chainID := deps.App.EvmKeeper.EthChainID(deps.Ctx)
				gasLimit := uint64(10)
				fees := sdk.NewCoins(sdk.NewInt64Coin("unibi", int64(gasLimit)))
				msg := buildEthMsg(chainID, gasLimit, deps.Sender.NibiruAddr.String(), nil)
				return buildTx(deps, true, msg, gasLimit, fees)
			},
			wantErr: "invalid From",
		},
		{
			name: "sad: tx with fee <> msg fee",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				chainID := deps.App.EvmKeeper.EthChainID(deps.Ctx)
				gasLimit := uint64(10)
				fees := sdk.NewCoins(sdk.NewInt64Coin("unibi", 5))
				msg := buildEthMsg(chainID, gasLimit, "", nil)
				return buildTx(deps, true, msg, gasLimit, fees)
			},
			wantErr: "invalid AuthInfo Fee Amount",
		},
		{
			name: "sad: tx with gas limit <> msg gas limit",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				chainID := deps.App.EvmKeeper.EthChainID(deps.Ctx)
				gasLimit := uint64(10)
				fees := sdk.NewCoins(sdk.NewInt64Coin("unibi", int64(gasLimit)))
				msg := buildEthMsg(chainID, gasLimit, "", nil)
				return buildTx(deps, true, msg, 5, fees)
			},
			wantErr: "invalid AuthInfo Fee GasLimit",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.NewStateDB()
			anteDec := evmante.NewEthValidateBasicDecorator(deps.App.AppKeepers.EvmKeeper)

			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			if tc.ctxSetup != nil {
				tc.ctxSetup(&deps)
			}
			var err error
			if tc.paramsSetup != nil {
				err = deps.EvmKeeper.SetParams(deps.Ctx, tc.paramsSetup(&deps))
			}

			if err == nil {
				_, err = anteDec.AnteHandle(
					deps.Ctx, tx, false, evmtest.NextNoOpAnteHandler,
				)
			}
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

func buildEthMsg(
	chainID *big.Int,
	gasLimit uint64,
	from string,
	to *common.Address,
) *evm.MsgEthereumTx {
	ethContractCreationTxParams := &evm.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: gasLimit,
		GasPrice: big.NewInt(1),
		To:       to,
	}
	tx := evm.NewTx(ethContractCreationTxParams)
	tx.From = from
	return tx
}

func buildTx(
	deps *evmtest.TestDeps,
	ethExtentions bool,
	msg sdk.Msg,
	gasLimit uint64,
	fees sdk.Coins,
) sdk.FeeTx {
	txBuilder, _ := deps.EncCfg.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
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

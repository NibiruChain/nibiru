package ante_test

import (
	"testing"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
)

func (s *Suite) TestOraclePostPriceTransactionsHaveFixedPrice() {
	priv1, addr := testutil.PrivKey()

	tests := []struct {
		name        string
		messages    []sdk.Msg
		expectedGas sdk.Gas
		expectedErr error
	}{
		{
			name: "Oracle Prevote Transaction",
			messages: []sdk.Msg{
				&oracletypes.MsgAggregateExchangeRatePrevote{
					Hash:      "dummyData",
					Feeder:    addr.String(),
					Validator: addr.String(),
				},
			},
			expectedGas: ante.OracleModuleTxGas,
			expectedErr: nil,
		},
		{
			name: "Oracle Vote Transaction",
			messages: []sdk.Msg{
				&oracletypes.MsgAggregateExchangeRateVote{
					Salt:          "dummySalt",
					ExchangeRates: "someData",
					Feeder:        addr.String(),
					Validator:     addr.String(),
				},
			},
			expectedGas: ante.OracleModuleTxGas,
			expectedErr: nil,
		},
		{
			name: "Two messages in a transaction, one of them is an oracle vote message should fail (with MsgAggregateExchangeRatePrevote)",
			messages: []sdk.Msg{
				&oracletypes.MsgAggregateExchangeRatePrevote{
					Hash:      "",
					Feeder:    addr.String(),
					Validator: addr.String(),
				},
				&bank.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 100)),
				},
			},
			expectedGas: 1042,
			expectedErr: sdkioerrors.Wrap(ante.ErrOracleAnte, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
		},
		{
			name: "Two messages in a transaction, one of them is an oracle vote message should fail (with MsgAggregateExchangeRatePrevote) permutation 2",
			messages: []sdk.Msg{
				&bank.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 100)),
				},
				&oracletypes.MsgAggregateExchangeRatePrevote{
					Hash:      "",
					Feeder:    addr.String(),
					Validator: addr.String(),
				},
			},
			expectedGas: 1042,
			expectedErr: sdkioerrors.Wrap(ante.ErrOracleAnte, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
		},
		{
			name: "Two messages in a transaction, one of them is an oracle vote message should fail (with MsgAggregateExchangeRateVote)",
			messages: []sdk.Msg{
				&oracletypes.MsgAggregateExchangeRateVote{
					Salt:          "dummySalt",
					ExchangeRates: "someData",
					Feeder:        addr.String(),
					Validator:     addr.String(),
				},
				&bank.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 100)),
				},
			},
			expectedGas: 1042,
			expectedErr: sdkioerrors.Wrap(ante.ErrOracleAnte, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
		},
		{
			name: "Two messages in a transaction, one of them is an oracle vote message should fail (with MsgAggregateExchangeRateVote) permutation 2",
			messages: []sdk.Msg{
				&bank.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 100)),
				},
				&oracletypes.MsgAggregateExchangeRateVote{
					Salt:          "dummySalt",
					ExchangeRates: "someData",
					Feeder:        addr.String(),
					Validator:     addr.String(),
				},
			},
			expectedGas: 1042,
			expectedErr: sdkioerrors.Wrap(ante.ErrOracleAnte, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
		},
		{
			name: "Two messages in a transaction, one is oracle vote, the other oracle pre vote: should work with fixed price",
			messages: []sdk.Msg{
				&oracletypes.MsgAggregateExchangeRatePrevote{
					Hash:      "",
					Feeder:    addr.String(),
					Validator: addr.String(),
				},
				&oracletypes.MsgAggregateExchangeRateVote{
					Salt:          "dummySalt",
					ExchangeRates: "someData",
					Feeder:        addr.String(),
					Validator:     addr.String(),
				},
			},
			expectedGas: ante.OracleModuleTxGas,
			expectedErr: nil,
		},
		{
			name: "Two messages in a transaction, one is oracle vote, the other oracle pre vote: should work with fixed price permutation 2",
			messages: []sdk.Msg{
				&oracletypes.MsgAggregateExchangeRateVote{
					Salt:          "dummySalt",
					ExchangeRates: "someData",
					Feeder:        addr.String(),
					Validator:     addr.String(),
				},
				&oracletypes.MsgAggregateExchangeRatePrevote{
					Hash:      "",
					Feeder:    addr.String(),
					Validator: addr.String(),
				},
			},
			expectedGas: ante.OracleModuleTxGas,
			expectedErr: nil,
		},
		{
			name: "Three messages in tx, two related to oracle, but other one is not: should fail",
			messages: []sdk.Msg{
				&oracletypes.MsgAggregateExchangeRateVote{
					Salt:          "dummySalt",
					ExchangeRates: "someData",
					Feeder:        addr.String(),
					Validator:     addr.String(),
				},
				&bank.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 100)),
				},
				&oracletypes.MsgAggregateExchangeRatePrevote{
					Hash:      "",
					Feeder:    addr.String(),
					Validator: addr.String(),
				},
			},
			expectedGas: 1042,
			expectedErr: sdkioerrors.Wrap(ante.ErrOracleAnte, "a transaction cannot have more than a single oracle vote and prevote message"),
		},
		{
			name: "Other two messages",
			messages: []sdk.Msg{
				&bank.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 100)),
				},
				&bank.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 200)),
				},
			},
			expectedGas: 67193,
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // setup
			txCfg := s.clientCtx.TxConfig
			s.txBuilder = txCfg.NewTxBuilder()

			// msg and signatures
			feeAmount := sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 150))
			gasLimit := testdata.NewTestGasLimit()
			s.txBuilder.SetFeeAmount(feeAmount)
			s.txBuilder.SetGasLimit(gasLimit)
			s.txBuilder.SetMemo("some memo")

			s.NoError(s.txBuilder.SetMsgs(tc.messages...))

			privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{11}, []uint64{0}
			tx, err := s.CreateTestTx(
				s.txBuilder,
				privs,
				accNums,
				accSeqs,
				s.ctx.ChainID(),
				txCfg,
			)
			s.NoErrorf(err, "tx: %v", tx)
			s.NoError(tx.ValidateBasic())
			s.ValidateTx(tx, s.T())

			err = testapp.FundAccount(
				s.app.BankKeeper, s.ctx, addr,
				sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 1000)),
			)
			s.Require().NoError(err)

			s.ctx, err = s.anteHandler(
				s.ctx,
				tx,
				/*simulate*/ true,
			)
			if tc.expectedErr != nil {
				s.Error(err)
				s.Contains(err.Error(), tc.expectedErr.Error())
			} else {
				s.NoError(err)
			}
			want := sdkmath.NewInt(int64(tc.expectedGas))
			got := sdkmath.NewInt(int64(s.ctx.GasMeter().GasConsumed()))
			s.Equal(want.String(), got.String())
		})
	}
}

func (s *Suite) ValidateTx(tx signing.Tx, t *testing.T) {
	memoTx, ok := tx.(sdk.TxWithMemo)
	if !ok {
		s.Fail(sdkioerrors.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type").Error(), "memoTx: %t", memoTx)
	}

	params := s.app.AccountKeeper.GetParams(s.ctx)
	s.EqualValues(256, params.MaxMemoCharacters)

	memoLen := len(memoTx.GetMemo())
	s.True(memoLen < int(params.MaxMemoCharacters))
}

func (s *Suite) TestZeroGasActors() {
	deps := evmtest.NewTestDeps()
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx(),
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(appconst.DENOM_UNIBI, sdk.NewInt(69_420))),
	))

	s.T().Logf("GIVEN: (sender, contract) NOT in ZeroGasActors")
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	deps.Commit()
	counterContractZeroGas := wasmContracts[1]
	zeroGasSender := evmtest.NewEthPrivAcc()
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders:   []string{zeroGasSender.NibiruAddr.String()},
		Contracts: []string{counterContractZeroGas.String()},
	})
	s.T().Logf("Zero gas actors { sender: %s, contract: %s }",
		zeroGasSender.NibiruAddr, counterContractZeroGas)

	wasmContracts = test.SetupWasmContracts(&deps, &s.Suite)
	deps.Commit()
	counterContractWithGasFees := wasmContracts[1]
	s.T().Logf("Normal actors { sender: %s, contract: %s }",
		deps.Sender.NibiruAddr, counterContractWithGasFees)

	blockHeader := tmproto.Header{
		Height:  deps.Ctx().BlockHeight(),
		ChainID: deps.Ctx().ChainID(),
		Time:    deps.Ctx().BlockTime(),
	}
	baseapp.SetChainID(deps.Ctx().ChainID())(deps.App.BaseApp)
	deps.App.BeginBlock(abci.RequestBeginBlock{Header: blockHeader})

	wasmMsg := wasm.RawContractMessage([]byte(`
	{
	  "increment": {}
	}
	`))
	err := wasmMsg.ValidateBasic()
	s.Require().NoError(err)

	unibi := func(x int64) sdk.Coins {
		return sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, x))
	}

	s.T().Log("tx has normal gas behavior if outside zero gas actors")
	{
		sender := deps.Sender
		contract := counterContractWithGasFees

		txCfg := deps.App.GetTxConfig()
		txBuilder := txCfg.NewTxBuilder()
		err = txBuilder.SetMsgs([]sdk.Msg{
			&wasm.MsgExecuteContract{
				Sender:   sender.NibiruAddr.String(),
				Contract: contract.String(),
				Msg:      wasmMsg,
				Funds:    sdk.Coins{},
			},
			&wasm.MsgExecuteContract{
				Sender:   sender.NibiruAddr.String(),
				Contract: contract.String(),
				Msg:      wasmMsg,
				Funds:    sdk.Coins{},
			},
		}...)
		s.Require().NoError(err)
		txBuilder.SetFeeAmount(unibi(500))
		txBuilder.SetGasLimit(250_000)

		privs := []cryptotypes.PrivKey{sender.PrivKey}
		accSender := deps.App.AccountKeeper.GetAccount(
			deps.Ctx(), sender.NibiruAddr,
		)
		accNums := []uint64{
			accSender.GetAccountNumber(),
		}
		accSeqs := []uint64{
			accSender.GetSequence(),
		}

		blockTx, err := s.CreateTestTx(
			txBuilder,
			privs,
			accNums,
			accSeqs,
			deps.Ctx().ChainID(),
			txCfg,
		)
		s.Require().NoError(err)

		s.T().Logf("blockHeader: %+v\n", blockHeader)
		baseapp.SetChainID(blockHeader.GetChainID())(deps.App.BaseApp)
		txBz, err := txCfg.TxEncoder()(blockTx)
		s.Require().NoError(err)
		deliverTxResp := deps.App.DeliverTx(abci.RequestDeliverTx{Tx: txBz})
		s.Require().True(deliverTxResp.IsOK(), "%#v", deliverTxResp)

		{
			r := deliverTxResp
			s.Greaterf(r.GasUsed, int64(100_000), `gasUsed="%d", resp: %#v`, r.GasUsed, r)
			s.EqualValuesf(250_000, r.GasWanted, `gasWanted="%d", resp: %#v`, r.GasWanted, r)
		}
	}

	s.T().Log("zero gas actors work as intended")
	{
		sender := zeroGasSender
		contract := counterContractZeroGas

		txCfg := deps.App.GetTxConfig()
		txBuilder := txCfg.NewTxBuilder()
		err = txBuilder.SetMsgs([]sdk.Msg{
			&wasm.MsgExecuteContract{
				Sender:   sender.NibiruAddr.String(),
				Contract: contract.String(),
				Msg:      wasmMsg,
				Funds:    sdk.Coins{},
			},
			&wasm.MsgExecuteContract{
				Sender:   sender.NibiruAddr.String(),
				Contract: contract.String(),
				Msg:      wasmMsg,
				Funds:    sdk.Coins{},
			},
		}...)
		s.Require().NoError(err)
		txBuilder.SetGasLimit(50_000)

		privs := []cryptotypes.PrivKey{sender.PrivKey}
		accSender := deps.App.AccountKeeper.GetAccount(
			deps.Ctx(), sender.NibiruAddr,
		)
		if accSender == nil {
			err = testapp.FundAccount(deps.App.BankKeeper,
				deps.Ctx(), sender.NibiruAddr, unibi(1),
			)
			s.Require().NoError(err)
			accSender = deps.App.AccountKeeper.GetAccount(
				deps.Ctx(), sender.NibiruAddr,
			)
			s.Require().NotNil(accSender)
		}
		accNums := []uint64{
			accSender.GetAccountNumber(),
		}
		accSeqs := []uint64{
			accSender.GetSequence(),
		}

		blockTx, err := s.CreateTestTx(
			txBuilder,
			privs,
			accNums,
			accSeqs,
			deps.Ctx().ChainID(),
			txCfg,
		)
		s.Require().NoError(err)

		balBefore := deps.App.BankKeeper.GetWeiBalance(deps.Ctx(), sender.NibiruAddr)

		txBz, err := txCfg.TxEncoder()(blockTx)
		s.Require().NoError(err)
		deliverTxResp := deps.App.DeliverTx(abci.RequestDeliverTx{Tx: txBz})
		s.Require().True(deliverTxResp.IsOK(), "%#v", deliverTxResp)

		{
			r := deliverTxResp
			s.EqualValuesf(0, r.GasUsed, `gasUsed="%d", resp: %#v`, r.GasUsed, r)
			s.EqualValuesf(-1, r.GasWanted, `gasWanted="%d", resp: %#v`, r.GasWanted, r)
			// Note that gasWanted == -1 indicates a fixed/zero gas path set by
			// the ante (FixedGasMeter), so the baseapp did not carry forward the
			// user-specified gas limit.
		}

		balAfter := deps.App.BankKeeper.GetWeiBalance(deps.Ctx(), sender.NibiruAddr)
		s.Equal(balAfter, balBefore, "expect zero gas charged")
	}

	deps.App.EndBlock(abci.RequestEndBlock{Height: deps.Ctx().BlockHeight()})
}

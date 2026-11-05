package ante_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/baseapp"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"

	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/signing"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	bank "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/v2/evm/precompile/test"
)

func (s *Suite) TestOraclePostPriceTransactionsHaveFixedPrice() {
	priv1, addr := testutil.NewPrivKey()

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

	s.T().Log("both contracts are registered with devgas")
	withdrawerWithGasFees := evmtest.NewEthPrivAcc().NibiruAddr
	withdrawerZeroGas := evmtest.NewEthPrivAcc().NibiruAddr
	for _, reg := range []struct{ contract, withdrawer sdk.AccAddress }{
		{counterContractWithGasFees, withdrawerWithGasFees},
		{counterContractZeroGas, withdrawerZeroGas},
	} {
		contract, withdrawer := reg.contract, reg.withdrawer
		deps.App.DevGasKeeper.SetFeeShare(deps.Ctx(), devgastypes.FeeShare{
			ContractAddress:   contract.String(),
			DeployerAddress:   deps.Sender.NibiruAddr.String(),
			WithdrawerAddress: withdrawer.String(),
		})
	}
	devgasParams := deps.App.DevGasKeeper.GetParams(deps.Ctx())
	s.Require().True(devgasParams.EnableFeeShare)
	feeCollector := deps.App.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	s.Require().NoError(testapp.FundModuleAccount(
		deps.App.BankKeeper, deps.Ctx(), authtypes.FeeCollectorName, unibi(1_000_000),
	))
	unibiBalance := func(addr sdk.AccAddress) sdkmath.Int {
		return deps.App.BankKeeper.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount
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
		s.True(unibiBalance(withdrawerWithGasFees).IsPositive(),
			"a fee-paying tx pays the devgas share")
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
		// A declared fee is never deducted from a zero-gas actor, so it must
		// not produce a devgas payout either.
		txBuilder.SetFeeAmount(unibi(500))

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
		feeCollectorBefore := unibiBalance(feeCollector)

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
		s.True(unibiBalance(withdrawerZeroGas).IsZero(),
			"a zero-gas actor tx pays no devgas share")
		s.Equal(feeCollectorBefore.String(), unibiBalance(feeCollector).String(),
			"the fee collector is unchanged by a zero-gas actor tx")
	}

	deps.App.EndBlock(abci.RequestEndBlock{Height: deps.Ctx().BlockHeight()})
}

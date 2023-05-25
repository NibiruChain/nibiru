package ante_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/errors"

	sdkerrors "cosmossdk.io/errors"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/ante"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

func (suite *AnteTestSuite) TestOraclePostPriceTransactionsHaveFixedPrice() {
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
			expectedGas: ante.OracleMessageGas,
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
			expectedGas: ante.OracleMessageGas,
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
				&types.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 100)),
				},
			},
			expectedGas: 5402,
			expectedErr: sdkerrors.Wrap(errors.ErrInvalidRequest, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
		},
		{
			name: "Two messages in a transaction, one of them is an oracle vote message should fail (with MsgAggregateExchangeRatePrevote) permutation 2",
			messages: []sdk.Msg{
				&types.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 100)),
				},
				&oracletypes.MsgAggregateExchangeRatePrevote{
					Hash:      "",
					Feeder:    addr.String(),
					Validator: addr.String(),
				},
			},
			expectedGas: 5402,
			expectedErr: sdkerrors.Wrap(errors.ErrInvalidRequest, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
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
				&types.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 100)),
				},
			},
			expectedGas: 5402,
			expectedErr: sdkerrors.Wrap(errors.ErrInvalidRequest, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
		},
		{
			name: "Two messages in a transaction, one of them is an oracle vote message should fail (with MsgAggregateExchangeRateVote) permutation 2",
			messages: []sdk.Msg{
				&types.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 100)),
				},
				&oracletypes.MsgAggregateExchangeRateVote{
					Salt:          "dummySalt",
					ExchangeRates: "someData",
					Feeder:        addr.String(),
					Validator:     addr.String(),
				},
			},
			expectedGas: 5402,
			expectedErr: sdkerrors.Wrap(errors.ErrInvalidRequest, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
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
			expectedGas: ante.OracleMessageGas,
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
			expectedGas: ante.OracleMessageGas,
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
				&types.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 100)),
				},
				&oracletypes.MsgAggregateExchangeRatePrevote{
					Hash:      "",
					Feeder:    addr.String(),
					Validator: addr.String(),
				},
			},
			expectedGas: 5402,
			expectedErr: sdkerrors.Wrap(errors.ErrInvalidRequest, "a transaction cannot have more than a single oracle vote and prevote message"),
		},
		{
			name: "Other two messages",
			messages: []sdk.Msg{
				&types.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 100)),
				},
				&types.MsgSend{
					FromAddress: addr.String(),
					ToAddress:   addr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 200)),
				},
			},
			expectedGas: 65951,
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		suite.T().Run(tc.name, func(t *testing.T) {
			suite.SetupTest() // setup
			suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

			// msg and signatures
			feeAmount := sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 150))
			gasLimit := testdata.NewTestGasLimit()
			suite.txBuilder.SetFeeAmount(feeAmount)
			suite.txBuilder.SetGasLimit(gasLimit)
			suite.txBuilder.SetMemo("some memo")

			suite.NoError(suite.txBuilder.SetMsgs(tc.messages...))

			privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{11}, []uint64{0}
			tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
			suite.NoErrorf(err, "tx: %v", tx)
			suite.NoError(tx.ValidateBasic())
			suite.ValidateTx(tx, suite.T())

			err = testapp.FundAccount(
				suite.app.BankKeeper, suite.ctx, addr,
				sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 1000)),
			)
			suite.Require().NoError(err)

			suite.ctx, err = suite.anteHandler(
				suite.ctx,
				tx,
				/*simulate*/ true,
			)
			if tc.expectedErr != nil {
				suite.Error(err)
				suite.Contains(err.Error(), tc.expectedErr.Error())
			} else {
				suite.NoError(err)
			}
			want := sdk.NewInt(int64(tc.expectedGas))
			got := sdk.NewInt(int64(suite.ctx.GasMeter().GasConsumed()))
			suite.Equal(want.String(), got.String())
		})
	}
}

func (s *AnteTestSuite) ValidateTx(tx signing.Tx, t *testing.T) {
	memoTx, ok := tx.(sdk.TxWithMemo)
	if !ok {
		s.Fail(sdkerrors.Wrap(errors.ErrTxDecode, "invalid transaction type").Error(), "memoTx: %t", memoTx)
	}

	params := s.app.AccountKeeper.GetParams(s.ctx)
	s.EqualValues(256, params.MaxMemoCharacters)

	memoLen := len(memoTx.GetMemo())
	s.True(memoLen < int(params.MaxMemoCharacters))
}

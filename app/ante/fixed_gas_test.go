package ante_test

import (
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/ante"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

func (suite *AnteTestSuite) TestOraclePostPriceTransactionsHaveFixedPrice() {
	priv1, _, addr := testdata.KeyTestPubAddr()

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
			expectedErr: sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
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
			expectedErr: sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
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
			expectedErr: sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
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
			expectedErr: sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages"),
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
			expectedErr: sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "a transaction cannot have more than a single oracle vote and prevote message"),
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
			expectedGas: 56814,
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
			suite.Require().NoError(suite.txBuilder.SetMsgs(tc.messages...))

			privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{12}, []uint64{0}
			tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
			suite.Require().NoError(err)

			err = simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr, sdk.NewCoins(sdk.NewInt64Coin(app.BondDenom, 1000)))
			if err != nil {
				return
			}

			suite.ctx, err = suite.anteHandler(suite.ctx, tx, false)
			if tc.expectedErr != nil {
				suite.Require().Contains(err.Error(), tc.expectedErr.Error())
			} else {
				suite.Require().NoError(err)
			}
			suite.Require().Equal(tc.expectedGas, suite.ctx.GasMeter().GasConsumed())
		})
	}
}

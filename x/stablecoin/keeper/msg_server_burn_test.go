package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil/sample"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgBurn_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name string
		msg  types.MsgBurn
		err  error
	}{
		{
			name: "invalid address",
			msg: types.MsgBurn{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: types.MsgBurn{
				Creator: sample.AccAddress().String(),
			},
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

//  TODO Write this test after we test the pricefeed keeper since there's a dependency
//  since there's a dependency between the two modules.
/*
func TestMsgBurnResponse_NotEnoughFunds(t *testing.T) {

	type TestCase struct {
		name        string
		accFunds    sdk.Coins
		msgBurn     types.MsgBurn
		msgResponse types.MsgBurnResponse
		govPrice    sdk.Dec
		collPrice   sdk.Dec
		err         error
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			acc := sample.AccAddress()
			oracle := sample.AccAddress()

			// Set prices for GOV and COLL
			priceKeeper := &matrixApp.PriceKeeper
			priceExpiry := time.Now().Add(10000)
			priceKeeper.SetPrice(ctx, oracle, "mtrx:ust", tc.govPrice, priceExpiry)
			priceKeeper.SetPrice(ctx, oracle, "ust:usdm", tc.collPrice, priceExpiry)

			err := simapp.FundAccount(matrixApp.BankKeeper, ctx, acc, tc.accFunds)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
		})
	}

	testCases := []TestCase{
		{
			name:     "Not enough GOV",
			accFunds: sdk.NewCoins(),
			msgBurn: types.MsgBurn{
				Creator: "invalid_address",
				Stable:  sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			msgResponse: types.MsgBurnResponse{
				Collateral: sdk.NewCoin("usdm", sdk.NewInt(0)),
				Gov:        sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name:     "Not enough COLL",
			accFunds: sdk.NewCoins(),
			msgBurn: types.MsgBurn{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			msgResponse: types.MsgBurnResponse{
				Collateral: sdk.NewCoin("usdm", sdk.NewInt(0)),
				Gov:        sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			err: nil,
		},
	}
	for _, test := range testCases {
		executeTest(t, test)
	}
}
*/

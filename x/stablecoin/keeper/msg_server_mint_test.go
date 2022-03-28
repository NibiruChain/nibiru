package keeper_test

import (
	"testing"
	"time"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/sample"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgMint_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  types.MsgMint
		err  error
	}{
		{
			name: "invalid address",
			msg: types.MsgMint{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: types.MsgMint{
				Creator: sample.AccAddress().String(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgMintResponse_NotEnoughFunds(t *testing.T) {

	type TestCase struct {
		name        string
		accFunds    sdk.Coins
		msgMint     types.MsgMint
		msgResponse types.MsgMintResponse
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
			// msgServer := keeper.NewMsgServerImpl(keeper.)
			// mintResponse := keeper.Mint(ctx, tc.msg)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
		})
	}

	tests := []TestCase{
		{
			name:     "Not enough GOV",
			accFunds: sdk.NewCoins(),
			msgMint: types.MsgMint{
				Creator: "invalid_address",
				Stable:  sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			msgResponse: types.MsgMintResponse{
				Stable: sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name:     "Not enough COLL",
			accFunds: sdk.NewCoins(),
			msgMint: types.MsgMint{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			msgResponse: types.MsgMintResponse{
				Stable: sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			err: nil,
		},
	}
	for _, test := range tests {
		executeTest(t, test)
	}
}

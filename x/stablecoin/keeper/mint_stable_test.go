package keeper_test

import (
	"fmt"
	"math"
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
		msg  types.MsgMintStable
		err  error
	}{
		{
			name: "invalid address",
			msg: types.MsgMintStable{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: types.MsgMintStable{
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

func TestMsgMintStableResponse_NotEnoughFunds(t *testing.T) {

	type TestCase struct {
		name        string
		accFunds    sdk.Coins
		msgMint     types.MsgMintStable
		msgResponse types.MsgMintStableResponse
		govPrice    sdk.Dec
		collPrice   sdk.Dec
		err         error
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			acc, _ := sdk.AccAddressFromBech32(tc.msgMint.Creator)
			oracle := sample.AccAddress()

			// Set prices for GOV and COLL
			priceKeeper := &matrixApp.PriceKeeper
			priceExpiry := time.Now().Add(time.Duration(math.Pow10(6)))
			postedPriceGov, err := priceKeeper.SetPrice(ctx, oracle, "mtrx:ust", tc.govPrice, priceExpiry)
			require.NoError(t, err)
			postedPriceColl, err := priceKeeper.SetPrice(ctx, oracle, "ust:usdm", tc.collPrice, priceExpiry)
			require.NoError(t, err)
			fmt.Println(postedPriceGov, postedPriceColl)

			// Fund account
			err = simapp.FundAccount(matrixApp.BankKeeper, ctx, acc, tc.accFunds)
			require.NoError(t, err)

			// Mint USDM
			goCtx := sdk.WrapSDKContext(ctx)
			mintStableResponse, err := matrixApp.StablecoinKeeper.MintStable(
				goCtx, &tc.msgMint)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			fmt.Println("prices: ", priceKeeper.GetCurrentPrices(ctx))
			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, mintStableResponse, tc.msgResponse, "mintStableResponse")
		})
	}

	tests := []TestCase{
		{
			name: "Not enough GOV",
			accFunds: sdk.NewCoins(
				sdk.NewCoin("ust", sdk.NewInt(1000000))),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       fmt.Errorf("all input prices are expired"),
		}, {
			name:     "Not enough COLL",
			accFunds: sdk.NewCoins(),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       nil,
		},
	}
	for _, test := range tests {
		executeTest(t, test)
	}
}

package keeper_test

import (
	"fmt"
	"testing"
	"time"

	ptypes "github.com/MatrixDao/matrix/x/pricefeed/types"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/sample"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgBurn_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name string
		msg  types.MsgBurnStable
		err  error
	}{
		{
			name: "invalid address",
			msg: types.MsgBurnStable{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: types.MsgBurnStable{
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

func TestMsgBurnResponse_NotEnoughFunds(t *testing.T) {

	type TestCase struct {
		name        string
		accFunds    sdk.Coins
		msgBurn     types.MsgBurnStable
		msgResponse types.MsgBurnStableResponse
		govPrice    sdk.Dec
		collPrice   sdk.Dec
		err         error
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			app, ctx := testutil.NewMatrixApp()
			acc := sample.AccAddress()
			oracle := sample.AccAddress()

			// Set prices for GOV and COLL
			mp := ptypes.Params{
				Markets: []ptypes.Market{
					{MarketID: "mtrx:ust", BaseAsset: "mtrx", QuoteAsset: "ust", Oracles: []sdk.AccAddress{}, Active: true},
					{MarketID: "ust:usdm", BaseAsset: "ust", QuoteAsset: "usdm", Oracles: []sdk.AccAddress{}, Active: true},
				},
			}
			keeper := app.PriceKeeper
			keeper.SetParams(ctx, mp)

			priceExpiry := time.Now().UTC().Add(1 * time.Hour)

			keeper.SetPrice(ctx, oracle, "mtrx:ust", tc.govPrice, priceExpiry)
			keeper.SetPrice(ctx, oracle, "ust:usdm", tc.collPrice, priceExpiry)
			simapp.FundAccount(app.BankKeeper, ctx, acc, tc.accFunds)

			// Mint USDM
			goCtx := sdk.WrapSDKContext(ctx)
			burnStableResponse, err := app.StablecoinKeeper.BurnStable(
				goCtx, &tc.msgBurn)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			fmt.Println("prices: ", keeper.GetCurrentPrices(ctx))
			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, burnStableResponse, tc.msgResponse, "mintStableResponse")
		})
	}

	testCases := []TestCase{
		{
			name:     "Not enough stable",
			accFunds: sdk.NewCoins(sdk.NewCoin("usdm", sdk.NewInt(10))),
			msgBurn: types.MsgBurnStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin("usdm", sdk.NewInt(11)),
			},
			msgResponse: types.MsgBurnStableResponse{
				Collateral: sdk.NewCoin("umtrx", sdk.NewInt(0)),
				Gov:        sdk.NewCoin("uust", sdk.NewInt(0)),
			},
			err: sdkerrors.ErrInvalidAddress,
		},
	}
	for _, test := range testCases {
		executeTest(t, test)
	}
}

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
		name         string
		accFunds     sdk.Coins
		msgBurn      types.MsgBurnStable
		msgResponse  types.MsgBurnStableResponse
		govPrice     sdk.Dec
		collPrice    sdk.Dec
		expectedPass bool
		err          string
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			app, ctx := testutil.NewMatrixApp()
			acc, _ := sdk.AccAddressFromBech32(tc.msgBurn.Creator)
			oracle := sample.AccAddress()

			// Set prices for GOV and COLL
			mp := ptypes.Params{
				Markets: []ptypes.Market{
					{MarketID: "mtrx:ust", BaseAsset: "ust", QuoteAsset: "mtrx",
						Oracles: []sdk.AccAddress{oracle}, Active: true},
					{MarketID: "usdm:ust", BaseAsset: "ust", QuoteAsset: "usdm",
						Oracles: []sdk.AccAddress{oracle}, Active: true},
				}}
			keeper := &app.PriceKeeper
			keeper.SetParams(ctx, mp)

			priceExpiry := ctx.BlockTime().Add(time.Hour)

			_, err := keeper.SetPrice(ctx, oracle, "mtrx:ust", tc.govPrice, priceExpiry)
			require.NoError(t, err)

			_, err = keeper.SetPrice(ctx, oracle, "usdm:ust", tc.collPrice, priceExpiry)
			require.NoError(t, err)

			for _, market := range mp.Markets {
				keeper.SetCurrentPrices(ctx, market.MarketID)
			}

			out := keeper.GetCurrentPrices(ctx)
			fmt.Println("prices: ", out)

			simapp.FundAccount(app.BankKeeper, ctx, acc, tc.accFunds)

			// Mint USDM
			goCtx := sdk.WrapSDKContext(ctx)

			burnStableResponse, err := app.StablecoinKeeper.BurnStable(
				goCtx, &tc.msgBurn)

			if !tc.expectedPass {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)

				return
			}
			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, burnStableResponse, tc.msgResponse, "mintStableResponse")
		})
	}

	testCases := []TestCase{
		{
			name:     "Not enough stable",
			accFunds: sdk.NewCoins(sdk.NewCoin("uusdm", sdk.NewInt(10))),
			msgBurn: types.MsgBurnStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin("uusdm", sdk.NewInt(11)),
			},
			msgResponse: types.MsgBurnStableResponse{
				Collateral: sdk.NewCoin("umtrx", sdk.NewInt(0)),
				Gov:        sdk.NewCoin("uust", sdk.NewInt(0)),
			},
			govPrice:     sdk.MustNewDecFromStr("10"),
			collPrice:    sdk.MustNewDecFromStr("1"),
			expectedPass: false,
			err:          "uusdm: Not enough balance",
		},
	}
	for _, test := range testCases {
		executeTest(t, test)
	}
}

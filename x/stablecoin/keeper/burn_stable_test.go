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
	stableDenom := "uusdm"
	govDenom := "umtrx"
	collDenom := "uust"
	govPricePool := "umtrx:uust"
	collPricePool := "uusdm:uust"

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

			matrixApp, ctx := testutil.NewMatrixApp()
			acc, _ := sdk.AccAddressFromBech32(tc.msgBurn.Creator)
			oracle := sample.AccAddress()

			// Set up markets for the pricefeed keeper.
			priceKeeper := &matrixApp.PriceKeeper
			pfParams := ptypes.Params{
				Markets: []ptypes.Market{
					{MarketID: govPricePool, BaseAsset: collDenom, QuoteAsset: govDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
					{MarketID: collPricePool, BaseAsset: collDenom, QuoteAsset: stableDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
				}}
			priceKeeper.SetParams(ctx, pfParams)

			// Post prices to each market with the oracle.
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			fmt.Println("Burn tests ------------")
			fmt.Println("Price expiry: ", priceExpiry.UTC().Unix())
			fmt.Println("ctx.BlockTime(): ", ctx.BlockTime().UTC().Unix())
			_, err := priceKeeper.SetPrice(
				ctx, oracle, govPricePool, tc.govPrice, priceExpiry,
			)
			require.NoError(t, err)
			_, err = priceKeeper.SetPrice(
				ctx, oracle, collPricePool, tc.collPrice, priceExpiry,
			)
			require.NoError(t, err)

			// Update the 'CurrentPrice' posted by the oracles.
			for _, market := range pfParams.Markets {
				err = priceKeeper.SetCurrentPrices(ctx, market.MarketID)
				require.NoError(t, err, "Error posting price for market: %d", market)
			}

			err = simapp.FundAccount(matrixApp.BankKeeper, ctx, acc, tc.accFunds)
			require.NoError(t, err)

			// Burn USDM -> Response contains GOV and COLL
			goCtx := sdk.WrapSDKContext(ctx)
			burnStableResponse, err := matrixApp.StablecoinKeeper.BurnStable(
				goCtx, &tc.msgBurn)

			if !tc.expectedPass {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)

				return
			}
			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, burnStableResponse, tc.msgResponse, "burnStableResponse")
		})
	}

	testCases := []TestCase{
		{
			name:     "Not enough stable",
			accFunds: sdk.NewCoins(sdk.NewCoin(stableDenom, sdk.NewInt(10))),
			msgBurn: types.MsgBurnStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(stableDenom, sdk.NewInt(9001)),
			},
			msgResponse: types.MsgBurnStableResponse{
				Collateral: sdk.NewCoin(govDenom, sdk.NewInt(0)),
				Gov:        sdk.NewCoin(collDenom, sdk.NewInt(0)),
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

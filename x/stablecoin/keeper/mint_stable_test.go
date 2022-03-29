package keeper_test

import (
	"fmt"
	"testing"
	"time"

	pricefeedTypes "github.com/MatrixDao/matrix/x/pricefeed/types"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"

	"github.com/MatrixDao/matrix/x/testutil/sample"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgMint_ValidateBasic(t *testing.T) {
	testCases := []struct {
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

// TODO: test (pricefeed/keeper): We need to test posted prices first.

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

			params := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: "mtrx:ust", BaseAsset: "ust", QuoteAsset: "mtrx",
						Oracles: []sdk.AccAddress{oracle}, Active: true},
					{MarketID: "usdm:ust", BaseAsset: "ust", QuoteAsset: "usdm",
						Oracles: []sdk.AccAddress{oracle}, Active: true},
				}}

			priceKeeper := &matrixApp.PriceKeeper
			priceKeeper.SetParams(ctx, params)

			priceExpiry := ctx.BlockTime().Add(time.Hour)

			_, err := priceKeeper.SetPrice(ctx, oracle, "mtrx:ust", tc.govPrice, priceExpiry)
			require.NoError(t, err)
			_, err = priceKeeper.SetPrice(ctx, oracle, "usdm:ust", tc.collPrice, priceExpiry)
			require.NoError(t, err)
			for _, market := range params.Markets {
				priceKeeper.SetCurrentPrices(ctx, market.MarketID)
			}

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

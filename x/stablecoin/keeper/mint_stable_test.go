package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/MatrixDao/matrix/x/common"
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

			// Set up markets for the pricefeed keeper.
			priceKeeper := &matrixApp.PriceKeeper
			pfParams := pricefeedTypes.Params{
				Markets: []pricefeedTypes.Market{
					{MarketID: common.GovPricePool, BaseAsset: common.CollDenom, QuoteAsset: common.GovDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
					{MarketID: common.CollStablePool, BaseAsset: common.CollDenom, QuoteAsset: common.StableDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
				}}
			priceKeeper.SetParams(ctx, pfParams)

			// Post prices to each market with the oracle.
			priceExpiry := ctx.BlockTime().Add(time.Hour)
			_, err := priceKeeper.SetPrice(
				ctx, oracle, common.GovPricePool, tc.govPrice, priceExpiry,
			)
			require.NoError(t, err)
			_, err = priceKeeper.SetPrice(
				ctx, oracle, common.CollStablePool, tc.collPrice, priceExpiry,
			)
			require.NoError(t, err)

			// Update the 'CurrentPrice' posted by the oracles.
			for _, market := range pfParams.Markets {
				err = priceKeeper.SetCurrentPrices(ctx, market.MarketID)
				require.NoError(t, err, "Error posting price for market: %d", market)
			}

			// Fund account
			err = simapp.FundAccount(matrixApp.BankKeeper, ctx, acc, tc.accFunds)
			require.NoError(t, err)

			// Mint USDM -> Response contains Stable (sdk.Coin)
			goCtx := sdk.WrapSDKContext(ctx)
			mintStableResponse, err := matrixApp.StablecoinKeeper.MintStable(
				goCtx, &tc.msgMint)
			if err == pricefeedTypes.ErrNoValidPrice {
				fmt.Println("Mint tests ------------")
				fmt.Println("Price expiry: ", priceExpiry.UTC().Unix())
				fmt.Println("ctx.BlockTime(): ", ctx.BlockTime().UTC().Unix())
				fmt.Println("Prices failed to post: ", priceKeeper.GetCurrentPrices(ctx))
			}

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)
			testutil.RequireEqualWithMessage(
				t, *mintStableResponse, tc.msgResponse, "mintStableResponse")
		})
	}

	testCases := []TestCase{
		{
			name: "User has no GOV",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(9001)),
				sdk.NewCoin(common.GovDenom, sdk.NewInt(0)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.StableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       types.NoCoinFound.Wrap(common.GovDenom),
		}, {
			name: "User has no COLL",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(0)),
				sdk.NewCoin(common.GovDenom, sdk.NewInt(9001)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.StableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       types.NoCoinFound.Wrap(common.CollDenom),
		},
		{
			name: "Not enough GOV",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(9001)),
				sdk.NewCoin(common.GovDenom, sdk.NewInt(1)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(1000)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.StableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err: types.NotEnoughBalance.Wrap(
				sdk.NewCoin(common.GovDenom, sdk.NewInt(1)).String()),
		}, {
			name: "Not enough COLL",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(1)),
				sdk.NewCoin(common.GovDenom, sdk.NewInt(9001)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.StableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err: types.NotEnoughBalance.Wrap(
				sdk.NewCoin(common.CollDenom, sdk.NewInt(1)).String()),
		},
		{
			name: "Successful mint",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(common.GovDenom, sdk.NewInt(9001)),
				sdk.NewCoin(common.CollDenom, sdk.NewInt(9001)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(common.StableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(common.StableDenom, sdk.NewInt(100)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       nil,
		},
	}
	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

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

func TestMsgMintStableResponse_NotEnoughFunds(t *testing.T) {
	stableDenom := "uusdm"
	govDenom := "umtrx"
	collDenom := "uust"
	govPricePool := "umtrx:uust"
	collPricePool := "uusdm:uust"

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
					{MarketID: govPricePool, BaseAsset: collDenom, QuoteAsset: govDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
					{MarketID: collPricePool, BaseAsset: collDenom, QuoteAsset: stableDenom,
						Oracles: []sdk.AccAddress{oracle}, Active: true},
				}}
			priceKeeper.SetParams(ctx, pfParams)

			// Post prices to each market with the oracle.
			priceExpiry := ctx.BlockTime().Add(time.Hour)
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
			name: "Not enough GOV",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(collDenom, sdk.NewInt(9001)),
				sdk.NewCoin(govDenom, sdk.NewInt(0)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(stableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(stableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       types.NoCoinFound.Wrap(govDenom),
		}, {
			name: "Not enough COLL",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(collDenom, sdk.NewInt(0)),
				sdk.NewCoin(govDenom, sdk.NewInt(9001)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(stableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(stableDenom, sdk.NewInt(0)),
			},
			govPrice:  sdk.MustNewDecFromStr("10"),
			collPrice: sdk.MustNewDecFromStr("1"),
			err:       types.NoCoinFound.Wrap(collDenom),
		}, {
			name: "Successful mint",
			accFunds: sdk.NewCoins(
				sdk.NewCoin(govDenom, sdk.NewInt(9001)),
				sdk.NewCoin(collDenom, sdk.NewInt(9001)),
			),
			msgMint: types.MsgMintStable{
				Creator: sample.AccAddress().String(),
				Stable:  sdk.NewCoin(stableDenom, sdk.NewInt(100)),
			},
			msgResponse: types.MsgMintStableResponse{
				Stable: sdk.NewCoin(stableDenom, sdk.NewInt(100)),
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

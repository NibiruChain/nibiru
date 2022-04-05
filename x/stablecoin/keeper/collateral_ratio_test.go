package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/common"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/sample"

	ptypes "github.com/MatrixDao/matrix/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSetCollRatio_Input(t *testing.T) {

	type TestCase struct {
		name         string
		inCollRatio  sdk.Dec
		expectedPass bool
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()
			stablecoinKeeper := &matrixApp.StablecoinKeeper

			err := stablecoinKeeper.SetCollRatio(ctx, tc.inCollRatio)
			if tc.expectedPass {
				require.NoError(
					t, err, "Error setting the CollRatio: %d", tc.inCollRatio)
				return
			}
			require.Error(t, err)
		})
	}

	testCases := []TestCase{
		{
			name:         "Upper bound of CollRatio",
			inCollRatio:  sdk.OneDec(),
			expectedPass: true,
		}, {
			name:         "Lower bound of CollRatio",
			inCollRatio:  sdk.ZeroDec(),
			expectedPass: true,
		}, {
			name:         "CollRatio above 100",
			inCollRatio:  sdk.MustNewDecFromStr("1.5"),
			expectedPass: false,
		}, {
			name:         "Negative CollRatio not allowed",
			inCollRatio:  sdk.OneDec().Neg(),
			expectedPass: false,
		},
	}
	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

func TestSetCollRatioUpdate(t *testing.T) {

	type TestCase struct {
		name              string
		inCollRatio       sdk.Dec
		price             sdk.Dec
		expectedCollRatio sdk.Dec
		expectedPass      bool
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {

			matrixApp, ctx := testutil.NewMatrixApp()

			stablecoinKeeper := &matrixApp.StablecoinKeeper
			priceKeeper := &matrixApp.PriceKeeper

			oracle := sample.AccAddress()
			markets := ptypes.NewParams([]ptypes.Market{

				{
					MarketID:   common.CollPricePool,
					BaseAsset:  common.CollDenom,
					QuoteAsset: common.StableDenom,
					Oracles:    []sdk.AccAddress{oracle},
					Active:     true,
				},
			})

			priceKeeper.SetParams(ctx, markets)

			stablecoinKeeper.SetCollRatio(ctx, tc.inCollRatio)
			priceKeeper.SimSetPrice(ctx, common.CollPricePool, tc.price)
			priceKeeper.SetCurrentPrices(ctx, common.CollPricePool)

			err := stablecoinKeeper.EvaluateCollRatio(ctx)
			if tc.expectedPass {
				require.NoError(
					t, err, "Error setting the CollRatio: %d", tc.inCollRatio)

				currCollRatio := stablecoinKeeper.GetCollRatio(ctx)
				require.Equal(t, tc.expectedCollRatio, currCollRatio)
				return
			}
			require.Error(t, err)
		})
	}

	testCases := []TestCase{
		{
			name:              "Price is higher than peg",
			inCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("1.1"),
			expectedCollRatio: sdk.MustNewDecFromStr("0.8025"),
			expectedPass:      true,
		},
		{
			name:              "Price is slightly higher than peg",
			inCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("1.00000001"),
			expectedCollRatio: sdk.MustNewDecFromStr("0.8"),
			expectedPass:      true,
		},
		{
			name:              "Price is slightly lower than peg",
			inCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("0.99999999991"),
			expectedCollRatio: sdk.MustNewDecFromStr("0.8"),
			expectedPass:      true,
		},
		{
			name:              "Price is lower than peg",
			inCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("0.9"),
			expectedCollRatio: sdk.MustNewDecFromStr("0.7975"),
			expectedPass:      true,
		},
	}
	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

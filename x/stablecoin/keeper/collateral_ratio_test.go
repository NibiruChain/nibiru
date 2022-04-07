package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"

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

func TestGetCollRatio_Input(t *testing.T) {

	testName := "GetCollRatio after setting default params returns expected value"
	t.Run(testName, func(t *testing.T) {

		matrixApp, ctx := testutil.NewMatrixApp()
		stablecoinKeeper := &matrixApp.StablecoinKeeper

		stablecoinKeeper.SetParams(ctx, types.DefaultParams())
		expectedCollRatioInt := sdk.NewInt(types.DefaultParams().CollRatio)

		outCollRatio := stablecoinKeeper.GetCollRatio(ctx)
		outCollRatioInt := outCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		require.EqualValues(t, expectedCollRatioInt, outCollRatioInt)
	})

	testName = "Setting to non-default value returns expected value"
	t.Run(testName, func(t *testing.T) {

		matrixApp, ctx := testutil.NewMatrixApp()
		stablecoinKeeper := &matrixApp.StablecoinKeeper

		expectedCollRatio := sdk.MustNewDecFromStr("0.5")
		expectedCollRatioInt := expectedCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		stablecoinKeeper.SetCollRatio(ctx, expectedCollRatio)

		outCollRatio := stablecoinKeeper.GetCollRatio(ctx)
		outCollRatioInt := outCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		require.EqualValues(t, expectedCollRatioInt, outCollRatioInt)
	})

}

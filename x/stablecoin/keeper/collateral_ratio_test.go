package keeper_test

import (
	"fmt"
	"testing"

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

	testName := "Get without set returns the default params"
	t.Run(testName, func(t *testing.T) {

		matrixApp, ctx := testutil.NewMatrixApp()
		// stablecoinKeeper := &matrixApp.StablecoinKeeper

		fmt.Println(matrixApp.StablecoinKeeper.GetParams(ctx))
		// fmt.Println(stablecoinKeeper.GetParams(ctx))
		// outCollRatio := stablecoinKeeper.GetCollRatio(ctx)
		// outCollRatioInt := outCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
		// defaultCollRatioInt := sdk.NewInt(types.DefaultParams().CollRatio)
		// require.True(t, outCollRatioInt == defaultCollRatioInt)
	})

	// testName = "Setting to a cust"
	// t.Run(testName, func(t *testing.T) {

	// 	matrixApp, ctx := testutil.NewMatrixApp()
	// 	stablecoinKeeper := &matrixApp.StablecoinKeeper

	// 	outCollRatio := stablecoinKeeper.GetCollRatio(ctx)
	// 	outCollRatioInt := outCollRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()
	// 	defaultCollRatioInt := sdk.NewInt(types.DefaultParams().CollRatio)
	// 	require.True(t, outCollRatioInt == defaultCollRatioInt)
	// })

}

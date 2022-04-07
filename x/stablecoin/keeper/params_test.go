package keeper_test

import (
	"fmt"
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGetAndSetParams(t *testing.T) {

	var testName string

	testName = "Get default Params"
	t.Run(testName, func(t *testing.T) {
		matrixApp, ctx := testutil.NewMatrixApp()
		stableKeeper := &matrixApp.StablecoinKeeper

		params := types.DefaultParams()
		stableKeeper.SetParams(ctx, params)

		require.EqualValues(t, params, stableKeeper.GetParams(ctx))
	})

	testName = "Get non-default params"
	t.Run(testName, func(t *testing.T) {
		matrixApp, ctx := testutil.NewMatrixApp()
		stableKeeper := &matrixApp.StablecoinKeeper

		collRatio := sdk.MustNewDecFromStr("0.5")
		params := types.NewParams(collRatio)
		stableKeeper.SetParams(ctx, params)

		require.EqualValues(t, params, stableKeeper.GetParams(ctx))
	})

	testName = "Calling Get without setting causes a panic"
	t.Run(testName, func(t *testing.T) {
		matrixApp, ctx := testutil.NewMatrixApp()
		stableKeeper := &matrixApp.StablecoinKeeper

		getParamsPanicRaised := func() error {
			defer func() error {
				if err := recover(); err != nil {
					return fmt.Errorf("panic occured: %s", err)
				} else {
					return nil
				}
			}()
			err := fmt.Errorf("panic occured: %d", stableKeeper.GetParams(ctx))
			return err
		}
		require.NoError(t, getParamsPanicRaised())
	})

}

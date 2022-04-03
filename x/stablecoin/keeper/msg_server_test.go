package keeper_test

import (
	"testing"

	sckeeper "github.com/MatrixDao/matrix/x/stablecoin/keeper"
	"github.com/MatrixDao/matrix/x/testutil"

	"github.com/stretchr/testify/require"
)

func TestNewMsgServerImpl(t *testing.T) {

	type TestCase struct {
		name   string
		keeper sckeeper.Keeper
		err    error
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			msgServer := sckeeper.NewMsgServerImpl(tc.keeper)
			require.True(t, msgServer != nil)
		})
	}

	matrixApp, _ := testutil.NewMatrixApp()
	testCases := []TestCase{
		{
			name:   "Default MatrixApp.StablecoinKeeper, should pass",
			keeper: matrixApp.StablecoinKeeper,
			err:    nil,
		},
	}

	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

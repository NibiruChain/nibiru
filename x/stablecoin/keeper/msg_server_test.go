package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	sckeeper "github.com/NibiruChain/nibiru/x/stablecoin/keeper"
)

func TestNewMsgServerImpl(t *testing.T) {
	type TestCase struct {
		name   string
		keeper sckeeper.Keeper
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			msgServer := sckeeper.NewMsgServerImpl(tc.keeper)
			require.True(t, msgServer != nil)
		})
	}

	nibiruApp, _ := testapp.NewNibiruTestAppAndContext(true)
	testCases := []TestCase{
		{
			name:   "Default NibiruApp.StablecoinKeeper, should pass",
			keeper: nibiruApp.StablecoinKeeper,
		},
	}

	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

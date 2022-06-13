package types_test

import (
	"reflect"
	"testing"

	sctypes "github.com/NibiruChain/nibiru/x/stablecoin/types"
	testutilapp "github.com/NibiruChain/nibiru/x/testutil/app"

	"github.com/stretchr/testify/assert"
)

// Verifies that the expected keepers (e.g. 'KeeperName') in x/stablecoin are
// implemented on the corresponding 'NibiruApp.KeeperName' field
func TestExpectedKeepers(t *testing.T) {
	type TestCase struct {
		name           string
		expectedKeeper interface{}
		appKeeper      interface{}
	}

	nibiruApp, _ := testutilapp.NewNibiruApp(true)
	testCases := []TestCase{
		{
			name:           "PricefeedKeeper from x/pricefeed",
			expectedKeeper: (*sctypes.PricefeedKeeper)(nil),
			appKeeper:      nibiruApp.PricefeedKeeper,
		},
		{
			name:           "BankKeeper from the cosmos-sdk",
			expectedKeeper: (*sctypes.BankKeeper)(nil),
			appKeeper:      nibiruApp.BankKeeper,
		},
		{
			name:           "AccountKeeper from the cosmos-sdk",
			expectedKeeper: (*sctypes.AccountKeeper)(nil),
			appKeeper:      nibiruApp.AccountKeeper,
		},
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			_interface := reflect.TypeOf(tc.expectedKeeper).Elem()
			isImplementingExpectedMethods := reflect.
				TypeOf(tc.appKeeper).Implements(_interface)
			assert.True(t, isImplementingExpectedMethods)
		})
	}

	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}

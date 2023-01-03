package types_test

import (
	"reflect"
	"testing"

	"github.com/NibiruChain/nibiru/simapp"

	"github.com/stretchr/testify/assert"

	sctypes "github.com/NibiruChain/nibiru/x/stablecoin/types"
)

// Verifies that the expected keepers (e.g. 'KeeperName') in x/stablecoin are
// implemented on the corresponding 'NibiruApp.KeeperName' field
func TestExpectedKeepers(t *testing.T) {
	type TestCase struct {
		name           string
		expectedKeeper interface{}
		appKeeper      interface{}
	}

	nibiruApp, _ := simapp.NewTestNibiruAppAndContext(true)
	testCases := []TestCase{
		{
			name:           "OracleKeeper from x/oracle",
			expectedKeeper: (*sctypes.OracleKeeper)(nil),
			appKeeper:      nibiruApp.OracleKeeper,
		},
		{
			name:           "bankKeeper from the cosmos-sdk",
			expectedKeeper: (*sctypes.BankKeeper)(nil),
			appKeeper:      nibiruApp.BankKeeper,
		},
		{
			name:           "accountKeeper from the cosmos-sdk",
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

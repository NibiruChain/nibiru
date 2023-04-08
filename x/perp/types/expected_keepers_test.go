package types_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

/*
TestExpectedKeepers verifies that the expected keeper interfaces in x/perp

	(see interfaces.go) are implemented on the corresponding app keeper,
	'NibiruApp.KeeperName'
*/
func TestExpectedKeepers(t *testing.T) {
	nibiruApp, _ := testapp.NewNibiruTestAppAndContext(true)
	testCases := []struct {
		name           string
		expectedKeeper interface{}
		appKeeper      interface{}
	}{
		{
			name:           "OracleKeeper from x/oracle",
			expectedKeeper: (*types.OracleKeeper)(nil),
			appKeeper:      nibiruApp.OracleKeeper,
		},
		{
			name:           "bankKeeper from the cosmos-sdk",
			expectedKeeper: (*types.BankKeeper)(nil),
			appKeeper:      nibiruApp.BankKeeper,
		},
		{
			name:           "accountKeeper from the cosmos-sdk",
			expectedKeeper: (*types.AccountKeeper)(nil),
			appKeeper:      nibiruApp.AccountKeeper,
		},
		{
			name:           "PerpAmmKeeper from x/perp/amm",
			expectedKeeper: (*types.PerpAmmKeeper)(nil),
			appKeeper:      nibiruApp.PerpAmmKeeper,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			_interface := reflect.TypeOf(tc.expectedKeeper).Elem()
			isImplementingExpectedMethods := reflect.
				TypeOf(tc.appKeeper).Implements(_interface)
			assert.True(t, isImplementingExpectedMethods)
		})
	}
}

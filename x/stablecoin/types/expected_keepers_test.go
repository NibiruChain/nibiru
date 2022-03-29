package types_test

import (
	"reflect"
	"testing"

	pftypes "github.com/MatrixDao/matrix/x/pricefeed/types"
	sctypes "github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

// test AccountKeeper
// test BankKeeper
// test PriceKeeper

type PriceKeeper interface {
	GetCurrentPrice(sdk.Context, string) (pftypes.CurrentPrice, error)
	GetCurrentPrices(ctx sdk.Context) pftypes.CurrentPrices
	GetRawPrices(ctx sdk.Context, marketId string) pftypes.PostedPrices
	GetMarket(ctx sdk.Context, marketID string) (pftypes.Market, bool)
	GetMarkets(ctx sdk.Context) pftypes.Markets
	GetOracle(ctx sdk.Context, marketID string, address sdk.AccAddress) (sdk.AccAddress, error)
	GetOracles(ctx sdk.Context, marketID string) ([]sdk.AccAddress, error)
	SetCurrentPrices(ctx sdk.Context, marketID string) error
}

// Verifies that the expected keepers (e.g. 'KeeperName') in x/stablecoin are
// implemented on the corresponding 'MatrixApp.KeeperName' field
func TestExpectedKeepers(t *testing.T) {
	type TestCase struct {
		name           string
		expectedKeeper interface{}
		appKeeper      interface{}
	}

	matrixApp, _ := testutil.NewMatrixApp()
	testCases := []TestCase{
		{
			name:           "PriceKeeper from x/pricefeed",
			expectedKeeper: (*sctypes.PriceKeeper)(nil),
			appKeeper:      matrixApp.PriceKeeper,
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

package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestCalcRemainMarginWithFundingPayment(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "get - no positions set raises vpool not found error",
			test: func() {
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				marginDelta := sdk.OneDec()
				_, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, types.Position{
						Pair: "osmo:nusd",
					}, marginDelta)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

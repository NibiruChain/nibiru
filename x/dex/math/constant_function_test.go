package math

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSolveConstantProductInvariantHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name           string
		xPrior         sdk.Dec
		xAfter         sdk.Dec
		xWeight        sdk.Dec
		yPrior         sdk.Dec
		yWeight        sdk.Dec
		expectedDeltaY sdk.Dec
	}{
		{
			// 100*(1-(100/200)^(.50/.50))
			name:           "simple numbers",
			xPrior:         sdk.NewDec(100),
			xAfter:         sdk.NewDec(200),
			xWeight:        sdk.NewDecWithPrec(5, 1),
			yPrior:         sdk.NewDec(100),
			yWeight:        sdk.NewDecWithPrec(5, 1),
			expectedDeltaY: sdk.NewDec(50),
		},
		{
			// 33*(1-(33/50)^(.50/.50))
			name:           "difficult numbers",
			xPrior:         sdk.NewDec(33),
			xAfter:         sdk.NewDec(50),
			xWeight:        sdk.NewDecWithPrec(5, 1),
			yPrior:         sdk.NewDec(33),
			yWeight:        sdk.NewDecWithPrec(5, 1),
			expectedDeltaY: sdk.NewDecWithPrec(1122, 2),
		},
		// TODO(https://github.com/NibiruChain/nibiru/issues/141): allow for uneven weights
		// {
		// 	// 44*(1-(86/35)^(.75/.25))
		// 	name:           "difficult numbers - uneven weights",
		// 	xPrior:         sdk.NewDec(86),
		// 	xAfter:         sdk.NewDec(35),
		// 	xWeight:        sdk.NewDecWithPrec(75, 2),
		// 	yPrior:         sdk.NewDec(44),
		// 	yWeight:        sdk.NewDecWithPrec(25, 2),
		// 	expectedDeltaY: sdk.NewDecWithPrec(-60874551603, 8),
		// },
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deltaY := SolveConstantProductInvariant(
				tc.xPrior, tc.xAfter, tc.xWeight, tc.yPrior, tc.yWeight)
			require.InDelta(t, tc.expectedDeltaY.MustFloat64(), deltaY.MustFloat64(), 0.0001)
		})
	}
}

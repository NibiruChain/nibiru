package types_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/stretchr/testify/require"
)

func TestPair(t *testing.T) {
	testCases := []struct {
		name   string
		pair   types.Pair
		proper bool
	}{
		{
			name:   "proper and improper order pairs are inverses-1",
			pair:   types.NewPair("atom", "osmo", nil, true),
			proper: true,
		},
		{
			name:   "proper and improper order pairs are inverses-2",
			pair:   types.NewPair("osmo", "atom", nil, true),
			proper: false,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			if tc.proper {
				require.True(t, tc.pair.IsProperOrder())
				require.EqualValues(t, tc.pair.Name(), tc.pair.AsString())
			} else {
				require.True(t, tc.pair.Inverse().IsProperOrder())
				require.EqualValues(t, tc.pair.Name(), tc.pair.Inverse().AsString())
			}

			require.True(t, true)
		})
	}
}

package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func TestGetLatestCumulativeFundingRate(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "happy path",
			test: func() {
				keeper, _, ctx := getKeeper(t)

				metadata := &types.PairMetadata{
					Pair: common.Pair_NIBI_NUSD,
					CumulativeFundingRates: []sdk.Dec{
						sdk.NewDec(1),
						sdk.NewDec(2), // returns the latest from the list
					},
				}
				setPairMetadata(keeper, ctx, *metadata)

				latestCumulativeFundingRate, err := keeper.
					getLatestCumulativeFundingRate(ctx, common.Pair_NIBI_NUSD)

				require.NoError(t, err)
				assert.Equal(t, sdk.NewDec(2), latestCumulativeFundingRate)
			},
		},
		{
			name: "uninitialized vpool has no metadata | fail",
			test: func() {
				perpKeeper, _, ctx := getKeeper(t)
				vpool := common.AssetPair{
					Token0: "xxx",
					Token1: "yyy",
				}
				lcpf, err := perpKeeper.getLatestCumulativeFundingRate(
					ctx, vpool)
				require.Error(t, err)
				assert.EqualValues(t, sdk.Dec{}, lcpf)
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

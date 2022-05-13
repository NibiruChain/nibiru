package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func setUp(t *testing.T) (perpKeeper Keeper, mocks mockedDependencies, ctx sdk.Context, traderAddr sdk.AccAddress, liquidatorAddr sdk.AccAddress) {
	perpKeeper, mocks, ctx = getKeeper(t)
	perpKeeper.SetParams(ctx, types.DefaultParams())

	traderAddr = sample.AccAddress()
	liquidatorAddr = sample.AccAddress()

	return
}

func TestLiquidate(t *testing.T) {
	testcases := []struct {
		name string
		test func()
	}{
		{
			name: "long position; negative pnl; margin below maintenance",
			test: func() {
				perpKeeper, mocks, ctx, traderAddr, liquidatorAddr := setUp(t)

				pairStr := "BTC:NUSD"
				pair := common.TokenPair(pairStr)

				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair:                       pair.String(),
					CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
				})

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair(pair),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair(pair),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(20), nil)

				mocks.mockVpoolKeeper.EXPECT().
					IsOverSpreadLimit(
						ctx,
						common.TokenPair(pair),
					).
					Return(false)

				t.Log("Opening the position")
				toLiquidatePosition := &types.Position{
					Address:      traderAddr.String(),
					Pair:         pairStr,
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}
				perpKeeper.SetPosition(ctx, pair, traderAddr.String(), toLiquidatePosition)

				t.Log("Liquidating the position")
				err := perpKeeper.Liquidate(ctx, pair, traderAddr, liquidatorAddr)
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

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

func Test_requireMoreMarginRatio(t *testing.T) {
	type test struct {
		marginRatio, baseMarginRatio sdk.Dec
		largerThanEqualTo            bool
		wantErr                      bool
	}

	cases := map[string]test{
		"ok - largeThanOrEqualTo true": {
			marginRatio:       sdk.NewDec(2),
			baseMarginRatio:   sdk.NewDec(1),
			largerThanEqualTo: true,
			wantErr:           false,
		},
		"ok - largerThanOrEqualTo false": {
			marginRatio:       sdk.NewDec(1),
			baseMarginRatio:   sdk.NewDec(2),
			largerThanEqualTo: false,
			wantErr:           false,
		},
		"fails - largerThanEqualTo true": {
			marginRatio:       sdk.NewDec(1),
			baseMarginRatio:   sdk.NewDec(2),
			largerThanEqualTo: true,
			wantErr:           true,
		},
		"fails - largerThanEqualTo false": {
			marginRatio:       sdk.NewDec(2),
			baseMarginRatio:   sdk.NewDec(1),
			largerThanEqualTo: false,
			wantErr:           true,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := requireMoreMarginRatio(tc.marginRatio, tc.baseMarginRatio, tc.largerThanEqualTo)
			switch {
			case tc.wantErr:
				if err == nil {
					t.Fatalf("expected error")
				}
			case !tc.wantErr:
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}
		})
	}
}

func TestKeeper_GetMarginRatio_Errors(t *testing.T) {
	tests := []struct {
		name     string
		position types.Position
	}{
		{
			"empty size position",
			types.Position{
				Size_: sdk.ZeroDec(),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			k, _, ctx := getKeeper(t)

			pos := tc.position

			_, err := k.GetMarginRatio(ctx, pos)
			require.EqualError(t, err, types.ErrPositionZero.Error())
		})
	}
}

func TestKeeper_GetMarginRatio(t *testing.T) {
	tests := []struct {
		name                string
		position            types.Position
		newPrice            sdk.Dec
		expectedMarginRatio sdk.Dec
	}{
		{
			"margin without price changes",
			types.Position{
				Address:                             sample.AccAddress().String(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(10),
				OpenNotional:                        sdk.NewDec(10),
				Margin:                              sdk.NewDec(1),
				LastUpdateCumulativePremiumFraction: sdk.OneDec(),
			},
			sdk.MustNewDecFromStr("10"),
			sdk.MustNewDecFromStr("0.1"),
		},
		{
			"margin with price changes",
			types.Position{
				Address:                             sample.AccAddress().String(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(10),
				OpenNotional:                        sdk.NewDec(10),
				Margin:                              sdk.NewDec(1),
				LastUpdateCumulativePremiumFraction: sdk.OneDec(),
			},
			sdk.MustNewDecFromStr("12"),
			sdk.MustNewDecFromStr("0.25"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			k, deps, ctx := getKeeper(t)

			t.Log("Mock vpool spot price")
			deps.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.TokenPair("BTC:NUSD"),
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.NewDec(10),
				).
				Return(tc.newPrice, nil)
			t.Log("Mock vpool twap")
			deps.mockVpoolKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					common.TokenPair("BTC:NUSD"),
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.NewDec(10),
					15*time.Minute,
				).
				Return(sdk.NewDec(10), nil)

			k.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       "BTC:NUSD",
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			marginRatio, err := k.GetMarginRatio(ctx, tc.position)
			require.NoError(t, err)
			require.Equal(t, tc.expectedMarginRatio, marginRatio)
		})
	}
}

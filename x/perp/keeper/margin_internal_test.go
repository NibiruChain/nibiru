package keeper

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
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
	k, _, ctx := getKeeper(t)

	pos := types.Position{
		Address:      sample.AccAddress().String(),
		Pair:         "BTC:NUSD",
		Size_:        sdk.NewDec(10),
		OpenNotional: sdk.NewDec(10),
		Margin:       sdk.NewDec(1),
	}

	_, err := k.GetMarginRatio(ctx, pos)
	require.EqualError(t, err, types.ErrPositionZero.Error())
}

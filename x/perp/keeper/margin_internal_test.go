package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"
)

func Test_requireMoreMarginRatio(t *testing.T) {
	type test struct {
		marginRatio, baseMarginRatio sdk.Int
		largerThanEqualTo            bool
		wantErr                      bool
	}

	cases := map[string]test{
		"ok - largeThanOrEqualTo true": {
			marginRatio:       sdk.NewInt(2),
			baseMarginRatio:   sdk.NewInt(1),
			largerThanEqualTo: true,
			wantErr:           false,
		},
		"ok - largerThanOrEqualTo false": {
			marginRatio:       sdk.NewInt(1),
			baseMarginRatio:   sdk.NewInt(2),
			largerThanEqualTo: false,
			wantErr:           false,
		},
		"fails - largerThanEqualTo true": {
			marginRatio:       sdk.NewInt(1),
			baseMarginRatio:   sdk.NewInt(2),
			largerThanEqualTo: true,
			wantErr:           true,
		},
		"fails - largerThanEqualTo false": {
			marginRatio:       sdk.NewInt(2),
			baseMarginRatio:   sdk.NewInt(1),
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

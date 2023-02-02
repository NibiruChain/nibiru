package types

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func TestPosition_Validate(t *testing.T) {
	type test struct {
		p       *Position
		wantErr bool
	}

	cases := map[string]test{
		"success": {
			p: &Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.MustNew("valid:pair"),
				Size_:                           sdk.MustNewDecFromStr("1000"),
				Margin:                          sdk.MustNewDecFromStr("1000"),
				OpenNotional:                    sdk.MustNewDecFromStr("1000"),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("1"),
				BlockNumber:                     0,
			},
			wantErr: false,
		},
		"bad trader address": {
			p:       &Position{TraderAddress: "invalid"},
			wantErr: true,
		},

		"bad pair": {
			p: &Position{
				TraderAddress: testutil.AccAddress().String(),
				Pair:          "",
			},
			wantErr: true,
		},

		"bad size": {
			p: &Position{
				TraderAddress: testutil.AccAddress().String(),
				Pair:          asset.MustNew("valid:pair"),
				Size_:         sdk.ZeroDec(),
			},
			wantErr: true,
		},

		"bad margin": {
			p: &Position{
				TraderAddress: testutil.AccAddress().String(),
				Pair:          asset.MustNew("valid:pair"),
				Size_:         sdk.MustNewDecFromStr("1000"),
				Margin:        sdk.MustNewDecFromStr("-1000"),
			},
			wantErr: true,
		},
		"bad open notional": {
			p: &Position{
				TraderAddress: testutil.AccAddress().String(),
				Pair:          asset.MustNew("valid:pair"),
				Size_:         sdk.MustNewDecFromStr("1000"),
				Margin:        sdk.MustNewDecFromStr("1000"),
				OpenNotional:  sdk.MustNewDecFromStr("-1000"),
			},
			wantErr: true,
		},

		"bad block number": {
			p: &Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.MustNew("valid:pair"),
				Size_:                           sdk.MustNewDecFromStr("1000"),
				Margin:                          sdk.MustNewDecFromStr("1000"),
				OpenNotional:                    sdk.MustNewDecFromStr("1000"),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("1"),
				BlockNumber:                     -1,
			},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.p.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected an error")
			} else if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

func TestPairMetadata_Validate(t *testing.T) {
	type test struct {
		p       *PairMetadata
		wantErr bool
	}

	cases := map[string]test{
		"success": {
			p: &PairMetadata{
				Pair:                            asset.MustNew("pair1:pair2"),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.1"),
			},
		},

		"invalid pair": {
			p:       &PairMetadata{},
			wantErr: true,
		},

		"invalid cumulative funding rate": {
			p: &PairMetadata{
				Pair:                            asset.MustNew("pair1:pair2"),
				LatestCumulativePremiumFraction: sdk.Dec{},
			},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.p.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected an error")
			} else if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

func BenchmarkPosition_Validate(b *testing.B) {
	t := &Position{
		TraderAddress:                   testutil.AccAddress().String(),
		Pair:                            asset.MustNew("valid:pair"),
		Size_:                           sdk.MustNewDecFromStr("1000"),
		Margin:                          sdk.MustNewDecFromStr("1000"),
		OpenNotional:                    sdk.MustNewDecFromStr("1000"),
		LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("1"),
		BlockNumber:                     0,
	}

	for i := 0; i < b.N; i++ {
		err := t.Validate()
		_ = err
	}
}

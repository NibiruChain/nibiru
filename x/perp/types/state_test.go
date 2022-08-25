package types

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"
)

func TestPosition_Validate(t *testing.T) {
	type test struct {
		p       *Position
		wantErr bool
	}

	cases := map[string]test{
		"success": {
			p: &Position{
				TraderAddress:                       sample.AccAddress().String(),
				Pair:                                common.MustNewAssetPair("valid:pair"),
				Size_:                               sdk.MustNewDecFromStr("1000"),
				Margin:                              sdk.MustNewDecFromStr("1000"),
				OpenNotional:                        sdk.MustNewDecFromStr("1000"),
				LastUpdateCumulativePremiumFraction: sdk.MustNewDecFromStr("1"),
				BlockNumber:                         0,
			},
			wantErr: false,
		},
		"bad trader address": {
			p:       &Position{TraderAddress: "invalid"},
			wantErr: true,
		},

		"bad pair": {
			p: &Position{
				TraderAddress: sample.AccAddress().String(),
				Pair:          common.AssetPair{},
			},
			wantErr: true,
		},

		"bad size": {
			p: &Position{
				TraderAddress: sample.AccAddress().String(),
				Pair:          common.MustNewAssetPair("valid:pair"),
				Size_:         sdk.ZeroDec(),
			},
			wantErr: true,
		},

		"bad margin": {
			p: &Position{
				TraderAddress: sample.AccAddress().String(),
				Pair:          common.MustNewAssetPair("valid:pair"),
				Size_:         sdk.MustNewDecFromStr("1000"),
				Margin:        sdk.MustNewDecFromStr("-1000"),
			},
			wantErr: true,
		},
		"bad open notional": {
			p: &Position{
				TraderAddress: sample.AccAddress().String(),
				Pair:          common.MustNewAssetPair("valid:pair"),
				Size_:         sdk.MustNewDecFromStr("1000"),
				Margin:        sdk.MustNewDecFromStr("1000"),
				OpenNotional:  sdk.MustNewDecFromStr("-1000"),
			},
			wantErr: true,
		},

		"bad block number": {
			p: &Position{
				TraderAddress:                       sample.AccAddress().String(),
				Pair:                                common.MustNewAssetPair("valid:pair"),
				Size_:                               sdk.MustNewDecFromStr("1000"),
				Margin:                              sdk.MustNewDecFromStr("1000"),
				OpenNotional:                        sdk.MustNewDecFromStr("1000"),
				LastUpdateCumulativePremiumFraction: sdk.MustNewDecFromStr("1"),
				BlockNumber:                         -1,
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

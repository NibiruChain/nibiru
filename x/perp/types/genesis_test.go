package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestGenesisState_Validate(t *testing.T) {
	type test struct {
		g       *GenesisState
		wantErr bool
	}

	cases := map[string]test{
		"success": {
			g: &GenesisState{
				Params: DefaultParams(),
				PairMetadata: []*PairMetadata{
					{
						Pair:                       common.MustNewAssetPair("pair1:pair2"),
						CumulativePremiumFractions: []sdk.Dec{sdk.MustNewDecFromStr("0.1")},
					},
				},
				Positions: []*Position{
					{
						TraderAddress:                       sample.AccAddress().String(),
						Pair:                                common.MustNewAssetPair("valid:pair"),
						Size_:                               sdk.MustNewDecFromStr("1000"),
						Margin:                              sdk.MustNewDecFromStr("1000"),
						OpenNotional:                        sdk.MustNewDecFromStr("1000"),
						LastUpdateCumulativePremiumFraction: sdk.MustNewDecFromStr("1"),
						BlockNumber:                         0,
					},
				},
				PrepaidBadDebts: []*PrepaidBadDebt{
					{
						Denom:  "pair",
						Amount: sdk.NewInt(10),
					},
				},
				WhitelistedAddresses: []string{sample.AccAddress().String()},
			},
			wantErr: false,
		},
		"bad params": {
			g:       &GenesisState{Params: Params{FeePoolFeeRatio: sdk.MustNewDecFromStr("-1.0")}},
			wantErr: true,
		},
		"bad position": {
			g: &GenesisState{
				Params: DefaultParams(),
				Positions: []*Position{
					{
						TraderAddress: sample.AccAddress().String(),
						Pair:          common.AssetPair{},
					},
				},
			},
			wantErr: true,
		},
		"bad whitelisted addr": {
			g:       &GenesisState{Params: DefaultParams(), WhitelistedAddresses: []string{"bad"}},
			wantErr: true,
		},
		"bad pair metadata": {
			g:       &GenesisState{Params: DefaultParams(), PairMetadata: []*PairMetadata{{Pair: common.AssetPair{}}}},
			wantErr: true,
		},

		"bad prepaid bad debt": {
			g: &GenesisState{Params: DefaultParams(), PrepaidBadDebts: []*PrepaidBadDebt{{
				Denom:  ":invalid:Denom",
				Amount: sdk.Int{},
			}}},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.g.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected an error")
			} else if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

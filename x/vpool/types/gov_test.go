package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestCreatePoolProposal_ValidateBasic(t *testing.T) {
	type test struct {
		m         *CreatePoolProposal
		expectErr bool
	}

	cases := map[string]test{
		"invalid pair": {&CreatePoolProposal{
			Title:       "add proposal",
			Description: "some weird description",
			Pair:        "invalidpair",
		}, true},

		"success": {
			m: &CreatePoolProposal{
				Title:             "add proposal",
				Description:       "some weird description",
				Pair:              "valid:pair",
				QuoteAssetReserve: sdk.NewDec(1_000_000),
				BaseAssetReserve:  sdk.NewDec(1_000_000),
				Config: VpoolConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.10"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.10"),
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
				},
			},
			expectErr: false,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.m.ValidateBasic()
			if err == nil && tc.expectErr {
				t.Fatal("error expected")
			} else if err != nil && !tc.expectErr {
				t.Fatal("unexpected error")
			}
		})
	}
}

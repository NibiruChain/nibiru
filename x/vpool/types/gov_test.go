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
		"invalid trade limit ratio < 0": {
			m: &CreatePoolProposal{
				Title:           "add proposal",
				Description:     "some weird description",
				Pair:            "valid:pair",
				TradeLimitRatio: sdk.NewDec(-1),
			},
			expectErr: true,
		},

		"invalid trade limit ratio > 1": {
			m: &CreatePoolProposal{
				Title:           "add proposal",
				Description:     "some weird description",
				Pair:            "valid:pair",
				TradeLimitRatio: sdk.NewDec(2),
			},
			expectErr: true,
		},

		"quote asset reserve 0": {
			m: &CreatePoolProposal{
				Title:             "add proposal",
				Description:       "some weird description",
				Pair:              "valid:pair",
				TradeLimitRatio:   sdk.MustNewDecFromStr("0.10"),
				QuoteAssetReserve: sdk.ZeroDec(),
			},
			expectErr: true,
		},

		"base asset reserve 0": {
			m: &CreatePoolProposal{
				Title:             "add proposal",
				Description:       "some weird description",
				Pair:              "valid:pair",
				TradeLimitRatio:   sdk.MustNewDecFromStr("0.10"),
				QuoteAssetReserve: sdk.NewDec(1_000_000),
				BaseAssetReserve:  sdk.ZeroDec(),
			},
			expectErr: true,
		},

		"fluctuation < 0": {
			m: &CreatePoolProposal{
				Title:                 "add proposal",
				Description:           "some weird description",
				Pair:                  "valid:pair",
				TradeLimitRatio:       sdk.MustNewDecFromStr("0.10"),
				QuoteAssetReserve:     sdk.NewDec(1_000_000),
				BaseAssetReserve:      sdk.NewDec(1_000_000),
				FluctuationLimitRatio: sdk.NewDec(-1),
			},
			expectErr: true,
		},

		"fluctuation > 1": {
			m: &CreatePoolProposal{
				Title:                 "add proposal",
				Description:           "some weird description",
				Pair:                  "valid:pair",
				TradeLimitRatio:       sdk.MustNewDecFromStr("0.10"),
				QuoteAssetReserve:     sdk.NewDec(1_000_000),
				BaseAssetReserve:      sdk.NewDec(1_000_000),
				FluctuationLimitRatio: sdk.NewDec(2),
			},
			expectErr: true,
		},

		"max oracle spread ratio < 0": {
			m: &CreatePoolProposal{
				Title:                 "add proposal",
				Description:           "some weird description",
				Pair:                  "valid:pair",
				TradeLimitRatio:       sdk.MustNewDecFromStr("0.10"),
				QuoteAssetReserve:     sdk.NewDec(1_000_000),
				BaseAssetReserve:      sdk.NewDec(1_000_000),
				FluctuationLimitRatio: sdk.MustNewDecFromStr("0.10"),
				MaxOracleSpreadRatio:  sdk.NewDec(-1),
			},
			expectErr: true,
		},
		"max oracle spread ratio > 1": {
			m: &CreatePoolProposal{
				Title:                 "add proposal",
				Description:           "some weird description",
				Pair:                  "valid:pair",
				TradeLimitRatio:       sdk.MustNewDecFromStr("0.10"),
				QuoteAssetReserve:     sdk.NewDec(1_000_000),
				BaseAssetReserve:      sdk.NewDec(1_000_000),
				FluctuationLimitRatio: sdk.MustNewDecFromStr("0.10"),
				MaxOracleSpreadRatio:  sdk.NewDec(2),
			},
			expectErr: true,
		},
		"maintenance ratio < 0": {
			m: &CreatePoolProposal{
				Title:                  "add proposal",
				Description:            "some weird description",
				Pair:                   "valid:pair",
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
				QuoteAssetReserve:      sdk.NewDec(1_000_000),
				BaseAssetReserve:       sdk.NewDec(1_000_000),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.10"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.10"),
				MaintenanceMarginRatio: sdk.NewDec(-1),
			},
			expectErr: true,
		},
		"maintenance ratio > 1": {
			m: &CreatePoolProposal{
				Title:                  "add proposal",
				Description:            "some weird description",
				Pair:                   "valid:pair",
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
				QuoteAssetReserve:      sdk.NewDec(1_000_000),
				BaseAssetReserve:       sdk.NewDec(1_000_000),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.10"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.10"),
				MaintenanceMarginRatio: sdk.NewDec(2),
			},
			expectErr: true,
		},

		"success": {
			m: &CreatePoolProposal{
				Title:                  "add proposal",
				Description:            "some weird description",
				Pair:                   "valid:pair",
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
				QuoteAssetReserve:      sdk.NewDec(1_000_000),
				BaseAssetReserve:       sdk.NewDec(1_000_000),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.10"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.10"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
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

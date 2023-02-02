package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func TestGenesisState_Validate(t *testing.T) {
	type test struct {
		genesis *GenesisState
		wantErr bool
	}

	cases := map[string]test{
		"success": {
			genesis: &GenesisState{
				Vpools: []Vpool{
					{
						Pair:              asset.MustNew("btc:usd"),
						BaseAssetReserve:  sdk.MustNewDecFromStr("100000"),
						QuoteAssetReserve: sdk.MustNewDecFromStr("100000"),
						Config: VpoolConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:              asset.MustNew("eth:usd"),
						BaseAssetReserve:  sdk.MustNewDecFromStr("100000"),
						QuoteAssetReserve: sdk.MustNewDecFromStr("100000"),
						Config: VpoolConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
				},
			},
			wantErr: false,
		},
		"invalid vpool": {
			genesis: &GenesisState{
				Vpools: []Vpool{
					{
						Pair:              asset.MustNew("btc:usd"),
						BaseAssetReserve:  sdk.MustNewDecFromStr("100000"),
						QuoteAssetReserve: sdk.MustNewDecFromStr("100000"),
						Config: VpoolConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:              asset.MustNew("invalid:usd"),
						BaseAssetReserve:  sdk.MustNewDecFromStr("100000"),
						QuoteAssetReserve: sdk.MustNewDecFromStr("100000"),
						Config: VpoolConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.5"),
							MaxLeverage:            sdk.MustNewDecFromStr("0"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
				},
			},
			wantErr: true,
		},
		"duplicate vpool": {
			genesis: &GenesisState{
				Vpools: []Vpool{
					{
						Pair:              asset.MustNew("btc:usd"),
						BaseAssetReserve:  sdk.MustNewDecFromStr("100000"),
						QuoteAssetReserve: sdk.MustNewDecFromStr("100000"),
						Config: VpoolConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:              asset.MustNew("eth:usd"),
						BaseAssetReserve:  sdk.MustNewDecFromStr("100000"),
						QuoteAssetReserve: sdk.MustNewDecFromStr("100000"),
						Config: VpoolConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:              asset.MustNew("eth:usd"),
						BaseAssetReserve:  sdk.MustNewDecFromStr("100000"),
						QuoteAssetReserve: sdk.MustNewDecFromStr("100000"),
						Config: VpoolConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			gotErr := tc.genesis.Validate()
			if tc.wantErr && gotErr == nil {
				t.Fatal("error expected")
			}
			if !tc.wantErr && gotErr != nil {
				t.Fatalf("unexpected error: %s", gotErr)
			}
		})
	}
}

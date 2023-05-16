package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
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
				Markets: []Market{
					{
						Pair:          asset.MustNewPair("btc:usd"),
						BaseReserve:   sdk.NewDec(100_000),
						QuoteReserve:  sdk.NewDec(100_000),
						SqrtDepth:     common.MustSqrtDec(sdk.NewDec(1e10)),
						PegMultiplier: sdk.OneDec(),
						Config: MarketConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:          asset.MustNewPair("eth:usd"),
						BaseReserve:   sdk.NewDec(100_000),
						QuoteReserve:  sdk.NewDec(100_000),
						SqrtDepth:     common.MustSqrtDec(sdk.NewDec(1e10)),
						PegMultiplier: sdk.OneDec(),
						Config: MarketConfig{
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
		"invalid market": {
			genesis: &GenesisState{
				Markets: []Market{
					{
						Pair:          asset.MustNewPair("btc:usd"),
						BaseReserve:   sdk.NewDec(100_000),
						QuoteReserve:  sdk.NewDec(100_000),
						SqrtDepth:     common.MustSqrtDec(sdk.NewDec(1e10)),
						PegMultiplier: sdk.OneDec(),
						Config: MarketConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:          asset.MustNewPair("invalid:usd"),
						BaseReserve:   sdk.NewDec(100_000),
						QuoteReserve:  sdk.NewDec(100_000),
						SqrtDepth:     common.MustSqrtDec(sdk.NewDec(1e10)),
						PegMultiplier: sdk.OneDec(),
						Config: MarketConfig{
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
		"duplicate market": {
			genesis: &GenesisState{
				Markets: []Market{
					{
						Pair:          asset.MustNewPair("btc:usd"),
						BaseReserve:   sdk.NewDec(100_000),
						QuoteReserve:  sdk.NewDec(100_000),
						PegMultiplier: sdk.OneDec(),
						SqrtDepth:     common.MustSqrtDec(sdk.NewDec(1e10)),
						Config: MarketConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:          asset.MustNewPair("eth:usd"),
						BaseReserve:   sdk.NewDec(100_000),
						QuoteReserve:  sdk.NewDec(100_000),
						PegMultiplier: sdk.OneDec(),
						SqrtDepth:     common.MustSqrtDec(sdk.NewDec(1e10)),
						Config: MarketConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:          asset.MustNewPair("eth:usd"),
						BaseReserve:   sdk.NewDec(100_000),
						QuoteReserve:  sdk.NewDec(100_000),
						PegMultiplier: sdk.OneDec(),
						SqrtDepth:     common.MustSqrtDec(sdk.NewDec(1e10)),
						Config: MarketConfig{
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
		"overflow": {
			genesis: &GenesisState{
				Markets: []Market{
					{
						Pair:          asset.MustNewPair("btc:usd"),
						BaseReserve:   sdk.MustNewDecFromStr("258359429617980926666593621001726127912.880790961315660013"), // will overflow
						QuoteReserve:  sdk.MustNewDecFromStr("258359429617980926666593621001726127912.880790961315660013"),
						SqrtDepth:     sdk.MustNewDecFromStr("258359429617980926666593621001726127912.880790961315660013"),
						PegMultiplier: sdk.OneDec(),
						Config: MarketConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:          asset.MustNewPair("eth:usd"),
						BaseReserve:   sdk.NewDec(100_000),
						QuoteReserve:  sdk.NewDec(100_000),
						SqrtDepth:     common.MustSqrtDec(sdk.NewDec(1e10)),
						PegMultiplier: sdk.OneDec(),
						Config: MarketConfig{
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
		"not overflow": {
			genesis: &GenesisState{
				Markets: []Market{
					{
						Pair:          asset.MustNewPair("btc:usd"),
						BaseReserve:   sdk.MustNewDecFromStr("258359429617980926666593621001726127812.880790961315660013"), // will not overflow
						QuoteReserve:  sdk.MustNewDecFromStr("258359429617980926666593621001726127812.880790961315660013"),
						SqrtDepth:     sdk.MustNewDecFromStr("258359429617980926666593621001726127812.880790961315660013"),
						PegMultiplier: sdk.OneDec(),
						Config: MarketConfig{
							FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.5"),
							MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.05"),
							MaxLeverage:            sdk.MustNewDecFromStr("10"),
							MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.5"),
							TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
						},
					},
					{
						Pair:          asset.MustNewPair("eth:usd"),
						BaseReserve:   sdk.NewDec(100_000),
						QuoteReserve:  sdk.NewDec(100_000),
						SqrtDepth:     common.MustSqrtDec(sdk.NewDec(1e10)),
						PegMultiplier: sdk.OneDec(),
						Config: MarketConfig{
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

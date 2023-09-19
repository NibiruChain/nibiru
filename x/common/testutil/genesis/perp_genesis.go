package genesis

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

var TEST_ENCODING_CONFIG = app.MakeEncodingConfigAndRegister()

func AddPerpV2Genesis(gen app.GenesisState) app.GenesisState {
	extraMarketAmms := map[asset.Pair]perpv2types.AmmMarket{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD): {
			Market: perpv2types.Market{
				Pair:                            asset.NewPair(denoms.BTC, denoms.NUSD),
				Version:                         1,
				Enabled:                         true,
				MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.04"),
				MaxLeverage:                     sdk.MustNewDecFromStr("20"),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
				EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
				LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
				PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
				FundingRateEpochId:              "30 min",
				MaxFundingRate:                  sdk.NewDec(1),
				TwapLookbackWindow:              time.Minute * 30,
				PrepaidBadDebt:                  sdk.NewInt64Coin(denoms.NUSD, 0),
			},
			Amm: perpv2types.AMM{
				Pair:            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Version:         1,
				BaseReserve:     sdk.NewDec(10e6),
				QuoteReserve:    sdk.NewDec(10e6),
				SqrtDepth:       sdk.NewDec(10e6),
				PriceMultiplier: sdk.NewDec(6_000),
				TotalLong:       sdk.ZeroDec(),
				TotalShort:      sdk.ZeroDec(),
			},
		},
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD): {
			Market: perpv2types.Market{
				Pair:                            asset.NewPair(denoms.ATOM, denoms.NUSD),
				Enabled:                         true,
				Version:                         1,
				MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:                     sdk.MustNewDecFromStr("15"),
				ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
				EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
				LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
				PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
				MaxFundingRate:                  sdk.NewDec(1),
				TwapLookbackWindow:              time.Minute * 30,
				PrepaidBadDebt:                  sdk.NewInt64Coin(denoms.NUSD, 0),
			},
			Amm: perpv2types.AMM{
				Pair:            asset.Registry.Pair(denoms.ATOM, denoms.NUSD),
				Version:         1,
				BaseReserve:     sdk.NewDec(10e6),
				QuoteReserve:    sdk.NewDec(10e6),
				SqrtDepth:       sdk.NewDec(10e6),
				PriceMultiplier: sdk.NewDec(6_000),
				TotalLong:       sdk.ZeroDec(),
				TotalShort:      sdk.ZeroDec(),
			},
		},
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD): {
			Market: perpv2types.Market{
				Pair:                            asset.NewPair(denoms.OSMO, denoms.NUSD),
				Enabled:                         true,
				Version:                         1,
				MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:                     sdk.MustNewDecFromStr("15"),
				ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
				EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
				LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
				PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
				FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
				MaxFundingRate:                  sdk.NewDec(1),
				TwapLookbackWindow:              time.Minute * 30,
				PrepaidBadDebt:                  sdk.NewInt64Coin(denoms.NUSD, 0),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			Amm: perpv2types.AMM{
				Pair:            asset.Registry.Pair(denoms.OSMO, denoms.NUSD),
				Version:         1,
				BaseReserve:     sdk.NewDec(10e6),
				QuoteReserve:    sdk.NewDec(10e6),
				SqrtDepth:       sdk.NewDec(10e6),
				PriceMultiplier: sdk.NewDec(6_000),
				TotalLong:       sdk.ZeroDec(),
				TotalShort:      sdk.ZeroDec(),
			},
		},
	}
	for pair, market := range START_MARKETS {
		extraMarketAmms[pair] = market
	}

	var marketsv2 []perpv2types.Market
	var ammsv2 []perpv2types.AMM
	var marketLastVersions []perpv2types.GenesisMarketLastVersion
	for _, marketAmm := range extraMarketAmms {
		marketsv2 = append(marketsv2, marketAmm.Market)
		ammsv2 = append(ammsv2, marketAmm.Amm)
		marketLastVersions = append(marketLastVersions, perpv2types.GenesisMarketLastVersion{
			Pair:    marketAmm.Market.Pair,
			Version: marketAmm.Market.Version,
		})
	}

	perpV2Gen := &perpv2types.GenesisState{
		Markets:            marketsv2,
		MarketLastVersions: marketLastVersions,
		Amms:               ammsv2,
		Positions:          []perpv2types.Position{},
		ReserveSnapshots:   []perpv2types.ReserveSnapshot{},
	}

	gen[perpv2types.ModuleName] = TEST_ENCODING_CONFIG.Marshaler.
		MustMarshalJSON(perpV2Gen)
	return gen
}

var START_MARKETS = map[asset.Pair]perpv2types.AmmMarket{
	asset.Registry.Pair(denoms.ETH, denoms.NUSD): {
		Market: perpv2types.Market{
			Pair:                            asset.Registry.Pair(denoms.ETH, denoms.NUSD),
			Enabled:                         true,
			Version:                         1,
			MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:                     sdk.MustNewDecFromStr("15"),
			LatestCumulativePremiumFraction: sdk.ZeroDec(),
			ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
			EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
			LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
			PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
			FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
			MaxFundingRate:                  sdk.NewDec(1),
			TwapLookbackWindow:              time.Minute * 30,
			PrepaidBadDebt:                  sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
		},
		Amm: perpv2types.AMM{
			Pair:            asset.Registry.Pair(denoms.ETH, denoms.NUSD),
			Version:         1,
			BaseReserve:     sdk.NewDec(10e6),
			QuoteReserve:    sdk.NewDec(10e6),
			SqrtDepth:       sdk.NewDec(10e6),
			PriceMultiplier: sdk.NewDec(6_000),
			TotalLong:       sdk.ZeroDec(),
			TotalShort:      sdk.ZeroDec(),
		},
	},
	asset.Registry.Pair(denoms.NIBI, denoms.NUSD): {
		Market: perpv2types.Market{
			Pair:                            asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
			Enabled:                         true,
			Version:                         1,
			MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.04"),
			MaxLeverage:                     sdk.MustNewDecFromStr("20"),
			LatestCumulativePremiumFraction: sdk.ZeroDec(),
			ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
			EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
			LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
			PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
			FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
			MaxFundingRate:                  sdk.NewDec(1),
			TwapLookbackWindow:              time.Minute * 30,
			PrepaidBadDebt:                  sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
		},
		Amm: perpv2types.AMM{
			Pair:            asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
			Version:         1,
			BaseReserve:     sdk.NewDec(10 * common.TO_MICRO),
			QuoteReserve:    sdk.NewDec(10 * common.TO_MICRO),
			SqrtDepth:       common.MustSqrtDec(sdk.NewDec(10 * common.TO_MICRO * 10 * common.TO_MICRO)),
			PriceMultiplier: sdk.NewDec(10),
			TotalLong:       sdk.ZeroDec(),
			TotalShort:      sdk.ZeroDec(),
		},
	},
}

func PerpV2Genesis() *perpv2types.GenesisState {
	return &perpv2types.GenesisState{
		Markets: []perpv2types.Market{
			{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Enabled:                         true,
				MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.04"),
				MaxLeverage:                     sdk.MustNewDecFromStr("20"),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
				EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
				LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
				PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
				FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
				MaxFundingRate:                  sdk.NewDec(1),
				TwapLookbackWindow:              time.Minute * 30,
				PrepaidBadDebt:                  sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
			},
		},
		MarketLastVersions: []perpv2types.GenesisMarketLastVersion{
			{
				Pair:    asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Version: 1,
			},
		},
		Amms: []perpv2types.AMM{
			{
				Pair:            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				BaseReserve:     sdk.NewDec(10 * common.TO_MICRO),
				QuoteReserve:    sdk.NewDec(10 * common.TO_MICRO),
				SqrtDepth:       common.MustSqrtDec(sdk.NewDec(10 * common.TO_MICRO * 10 * common.TO_MICRO)),
				PriceMultiplier: sdk.NewDec(10),
				TotalLong:       sdk.ZeroDec(),
				TotalShort:      sdk.ZeroDec(),
			},
		},
		Positions:        []perpv2types.Position{},
		ReserveSnapshots: []perpv2types.ReserveSnapshot{},
	}
}

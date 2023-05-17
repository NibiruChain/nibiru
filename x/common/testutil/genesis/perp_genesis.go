package genesis

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"

	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"

	perpammtypes "github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
	perptypes "github.com/NibiruChain/nibiru/x/perp/v1/types"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

var (
	TEST_ENCODING_CONFIG = app.MakeTestEncodingConfig()
)

func AddPerpGenesis(gen app.GenesisState) app.GenesisState {
	gen[perpammtypes.ModuleName] = TEST_ENCODING_CONFIG.Marshaler.
		MustMarshalJSON(PerpAmmGenesis())
	gen[perptypes.ModuleName] = TEST_ENCODING_CONFIG.Marshaler.
		MustMarshalJSON(PerpGenesis())
	return gen
}

func AddPerpV2Genesis(gen app.GenesisState) app.GenesisState {
	extraMarkets := map[asset.Pair]perpammtypes.Market{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD): {
			Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			BaseReserve:   sdk.NewDec(10e6),
			QuoteReserve:  sdk.NewDec(10e6),
			SqrtDepth:     sdk.NewDec(10e6),
			PegMultiplier: sdk.NewDec(6_000),
			TotalLong:     sdk.ZeroDec(),
			TotalShort:    sdk.ZeroDec(),
			Config: perpammtypes.MarketConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.04"),
				MaxLeverage:            sdk.MustNewDecFromStr("20"),
			},
		},
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD): {
			Pair:          asset.Registry.Pair(denoms.ATOM, denoms.NUSD),
			BaseReserve:   sdk.NewDec(10 * common.TO_MICRO),
			QuoteReserve:  sdk.NewDec(10 * common.TO_MICRO),
			SqrtDepth:     common.MustSqrtDec(sdk.NewDec(10 * common.TO_MICRO)),
			TotalLong:     sdk.ZeroDec(),
			TotalShort:    sdk.ZeroDec(),
			PegMultiplier: sdk.NewDec(6_000),
			Config: perpammtypes.MarketConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("1"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
				MaxOracleSpreadRatio:   sdk.OneDec(),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
		},
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD): {
			Pair:          asset.Registry.Pair(denoms.OSMO, denoms.NUSD),
			BaseReserve:   sdk.NewDec(10 * common.TO_MICRO),
			QuoteReserve:  sdk.NewDec(10 * common.TO_MICRO),
			SqrtDepth:     common.MustSqrtDec(sdk.NewDec(10 * common.TO_MICRO)),
			TotalLong:     sdk.ZeroDec(),
			TotalShort:    sdk.ZeroDec(),
			PegMultiplier: sdk.NewDec(6_000),
			Config: perpammtypes.MarketConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
				MaxOracleSpreadRatio:   sdk.OneDec(),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
		},
	}
	for pair, market := range START_MARKETS {
		extraMarkets[pair] = market
	}

	var marketsv2 []perpv2types.Market
	var ammsv2 []perpv2types.AMM
	defaultParams := perptypes.DefaultParams()
	for pair, market := range extraMarkets {
		marketsv2 = append(marketsv2, perpv2types.Market{
			Pair:                            pair,
			Enabled:                         true,
			PriceFluctuationLimitRatio:      market.Config.FluctuationLimitRatio,
			MaintenanceMarginRatio:          market.Config.MaintenanceMarginRatio,
			MaxLeverage:                     market.Config.MaxLeverage,
			LatestCumulativePremiumFraction: sdk.ZeroDec(),
			ExchangeFeeRatio:                defaultParams.FeePoolFeeRatio,
			EcosystemFundFeeRatio:           defaultParams.EcosystemFundFeeRatio,
			LiquidationFeeRatio:             defaultParams.LiquidationFeeRatio,
			PartialLiquidationRatio:         defaultParams.PartialLiquidationRatio,
			FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
			TwapLookbackWindow:              time.Minute * 30,
			PrepaidBadDebt:                  sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()),
		})
		ammsv2 = append(ammsv2, perpv2types.AMM{
			Pair:            pair,
			BaseReserve:     market.BaseReserve,
			QuoteReserve:    market.QuoteReserve,
			SqrtDepth:       market.SqrtDepth,
			PriceMultiplier: market.PegMultiplier,
			TotalLong:       market.TotalLong,
			TotalShort:      market.TotalShort,
		})
	}

	perpV2Gen := &perpv2types.GenesisState{
		Markets:          marketsv2,
		Amms:             ammsv2,
		Positions:        []perpv2types.Position{},
		ReserveSnapshots: []perpv2types.ReserveSnapshot{},
	}

	gen[perpv2types.ModuleName] = TEST_ENCODING_CONFIG.Marshaler.
		MustMarshalJSON(perpV2Gen)
	return gen
}

func AddOracleGenesis(gen app.GenesisState) app.GenesisState {
	gen[oracletypes.ModuleName] = TEST_ENCODING_CONFIG.Marshaler.
		MustMarshalJSON(OracleGenesis())
	return gen
}

var START_MARKETS = map[asset.Pair]perpammtypes.Market{
	asset.Registry.Pair(denoms.ETH, denoms.NUSD): {
		Pair:          asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		BaseReserve:   sdk.NewDec(10 * common.TO_MICRO),
		QuoteReserve:  sdk.NewDec(10 * common.TO_MICRO),
		SqrtDepth:     common.MustSqrtDec(sdk.NewDec(10 * common.TO_MICRO * 10 * common.TO_MICRO)),
		PegMultiplier: sdk.NewDec(6_000),
		TotalLong:     sdk.ZeroDec(),
		TotalShort:    sdk.ZeroDec(),
		Config: perpammtypes.MarketConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	},
	asset.Registry.Pair(denoms.NIBI, denoms.NUSD): {
		Pair:          asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
		BaseReserve:   sdk.NewDec(10 * common.TO_MICRO),
		QuoteReserve:  sdk.NewDec(10 * common.TO_MICRO),
		SqrtDepth:     common.MustSqrtDec(sdk.NewDec(10 * common.TO_MICRO * 10 * common.TO_MICRO)),
		PegMultiplier: sdk.NewDec(10),
		TotalLong:     sdk.ZeroDec(),
		TotalShort:    sdk.ZeroDec(),
		Config: perpammtypes.MarketConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.04"),
			MaxLeverage:            sdk.MustNewDecFromStr("20"),
		},
	},
}

func PerpGenesis() *perptypes.GenesisState {
	gen := perptypes.DefaultGenesis()
	var pairMetadata []perptypes.PairMetadata
	for pair := range START_MARKETS {
		pairMetadata = append(
			pairMetadata,
			perptypes.PairMetadata{
				Pair:                            pair,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
		)
	}
	gen.PairMetadata = pairMetadata
	return gen
}

func PerpV2Genesis() *perpv2types.GenesisState {
	markets := make(map[asset.Pair]perpammtypes.Market)

	extraMarkets := map[asset.Pair]perpammtypes.Market{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD): {
			Pair:          asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
			BaseReserve:   sdk.NewDec(10 * common.TO_MICRO),
			QuoteReserve:  sdk.NewDec(10 * common.TO_MICRO),
			SqrtDepth:     common.MustSqrtDec(sdk.NewDec(10 * common.TO_MICRO * 10 * common.TO_MICRO)),
			PegMultiplier: sdk.NewDec(10),
			TotalLong:     sdk.ZeroDec(),
			TotalShort:    sdk.ZeroDec(),
			Config: perpammtypes.MarketConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.04"),
				MaxLeverage:            sdk.MustNewDecFromStr("20"),
			},
		},
	}

	for pair, market := range extraMarkets {
		markets[pair] = market
	}

	var marketsv2 []perpv2types.Market
	var ammsv2 []perpv2types.AMM
	defaultParams := perptypes.DefaultParams()
	for pair, market := range markets {
		marketsv2 = append(marketsv2, perpv2types.Market{
			Pair:                            pair,
			Enabled:                         true,
			PriceFluctuationLimitRatio:      market.Config.FluctuationLimitRatio,
			MaintenanceMarginRatio:          market.Config.MaintenanceMarginRatio,
			MaxLeverage:                     market.Config.MaxLeverage,
			LatestCumulativePremiumFraction: sdk.ZeroDec(),
			ExchangeFeeRatio:                defaultParams.FeePoolFeeRatio,
			EcosystemFundFeeRatio:           defaultParams.EcosystemFundFeeRatio,
			LiquidationFeeRatio:             defaultParams.LiquidationFeeRatio,
			PartialLiquidationRatio:         defaultParams.PartialLiquidationRatio,
			FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
			TwapLookbackWindow:              time.Minute * 30,
			PrepaidBadDebt:                  sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
		})
		ammsv2 = append(ammsv2, perpv2types.AMM{
			Pair:            pair,
			BaseReserve:     market.BaseReserve,
			QuoteReserve:    market.QuoteReserve,
			SqrtDepth:       market.SqrtDepth,
			PriceMultiplier: market.PegMultiplier,
			TotalLong:       market.TotalLong,
			TotalShort:      market.TotalShort,
		})
	}

	gen := &perpv2types.GenesisState{
		Markets:          marketsv2,
		Amms:             ammsv2,
		Positions:        []perpv2types.Position{},
		ReserveSnapshots: []perpv2types.ReserveSnapshot{},
	}
	return gen
}

func PerpAmmGenesis() *perpammtypes.GenesisState {
	perpAmmGenesis := perpammtypes.DefaultGenesis()
	perpAmmGenesis.Markets = []perpammtypes.Market{
		START_MARKETS[asset.Registry.Pair(denoms.ETH, denoms.NUSD)],
		START_MARKETS[asset.Registry.Pair(denoms.NIBI, denoms.NUSD)],
	}
	return perpAmmGenesis
}

func OracleGenesis() *oracletypes.GenesisState {
	oracleGenesis := oracletypes.DefaultGenesisState()
	oracleGenesis.ExchangeRates = []oracletypes.ExchangeRateTuple{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: sdk.NewDec(1_000)},
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: sdk.NewDec(10)},
	}
	oracleGenesis.Params.VotePeriod = 1_000

	return oracleGenesis
}

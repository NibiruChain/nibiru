package genesis

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
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

func AddOracleGenesis(gen app.GenesisState) app.GenesisState {
	gen[oracletypes.ModuleName] = TEST_ENCODING_CONFIG.Marshaler.
		MustMarshalJSON(OracleGenesis())
	return gen
}

var START_MARKETS = map[asset.Pair]perpammtypes.Market{
	asset.Registry.Pair(denoms.ETH, denoms.NUSD): {
		Pair:              asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		BaseAssetReserve:  sdk.NewDec(10 * common.TO_MICRO),
		QuoteAssetReserve: sdk.NewDec(60_000 * common.TO_MICRO),
		SqrtDepth:         common.MustSqrtDec(sdk.NewDec(600_000 * common.TO_MICRO * common.TO_MICRO)),
		Bias:              sdk.ZeroDec(),
		PegMultiplier:     sdk.OneDec(),
		Config: perpammtypes.MarketConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	},
	asset.Registry.Pair(denoms.NIBI, denoms.NUSD): {
		Pair:              asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
		BaseAssetReserve:  sdk.NewDec(500_000),
		QuoteAssetReserve: sdk.NewDec(5 * common.TO_MICRO),
		SqrtDepth:         common.MustSqrtDec(sdk.NewDec(5 * 500_000 * common.TO_MICRO)),
		Bias:              sdk.ZeroDec(),
		PegMultiplier:     sdk.OneDec(),
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

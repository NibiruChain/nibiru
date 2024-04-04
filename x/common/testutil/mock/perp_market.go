package mock

import (
	sdkmath "cosmossdk.io/math"
	time "time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	asset "github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// default market with sensible values for tests
func TestMarket() *types.Market {
	return &types.Market{
		Pair:                            asset.NewPair(denoms.BTC, denoms.NUSD),
		Enabled:                         true,
		Version:                         1,
		MaintenanceMarginRatio:          sdkmath.LegacyMustNewDecFromStr("0.0625"),
		MaxLeverage:                     sdkmath.LegacyMustNewDecFromStr("10"),
		LatestCumulativePremiumFraction: sdkmath.LegacyZeroDec(),
		ExchangeFeeRatio:                sdkmath.LegacyMustNewDecFromStr("0.0010"),
		EcosystemFundFeeRatio:           sdkmath.LegacyMustNewDecFromStr("0.0010"),
		LiquidationFeeRatio:             sdkmath.LegacyMustNewDecFromStr("0.0005"),
		PartialLiquidationRatio:         sdkmath.LegacyMustNewDecFromStr("0.5"),
		FundingRateEpochId:              "30 min",
		MaxFundingRate:                  sdkmath.LegacyNewDec(1),
		TwapLookbackWindow:              time.Minute * 30,
		PrepaidBadDebt:                  sdk.NewInt64Coin(denoms.NUSD, 0),
		OraclePair:                      asset.NewPair(denoms.BTC, denoms.USD),
	}
}

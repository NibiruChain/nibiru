package mock

import (
	time "time"

	asset "github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// default market with sensible values for tests
func TestMarket() *v2types.Market {
	return &v2types.Market{
		Pair:                            asset.NewPair(denoms.BTC, denoms.NUSD),
		Enabled:                         true,
		PriceFluctuationLimitRatio:      sdk.MustNewDecFromStr("0.1"),
		MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.0625"),
		MaxLeverage:                     sdk.MustNewDecFromStr("10"),
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
		EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
		LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0005"),
		PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
		FundingRateEpochId:              "30 min",
		TwapLookbackWindow:              time.Minute * 30,
		PrepaidBadDebt:                  sdk.NewInt64Coin(denoms.NUSD, 0),
	}
}

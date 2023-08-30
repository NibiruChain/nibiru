package mock

import (
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
		MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.0625"),
		MaxLeverage:                     sdk.MustNewDecFromStr("10"),
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
		EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
		LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0005"),
		PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
		FundingRateEpochId:              "30 min",
		MaxFundingRate:                  sdk.NewDec(1),
		TwapLookbackWindow:              time.Minute * 30,
		PrepaidBadDebt:                  sdk.NewInt64Coin(denoms.NUSD, 0),
	}
}

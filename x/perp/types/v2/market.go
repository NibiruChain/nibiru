package v2

import (
	fmt "fmt"

	"github.com/NibiruChain/nibiru/x/common/asset"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func isPercent(v sdk.Dec) bool {
	return v.GTE(sdk.ZeroDec()) && v.LTE(sdk.OneDec())
}

func (market *Market) Validate() error {
	if !isPercent(market.PriceFluctuationLimitRatio) {
		return fmt.Errorf("fluctuation limit ratio must be 0 <= ratio <= 1")
	}

	if !isPercent(market.MaintenanceMarginRatio) {
		return fmt.Errorf("maintenance margin ratio ratio must be 0 <= ratio <= 1")
	}

	if !isPercent(market.EcosystemFundFeeRatio) {
		return fmt.Errorf("ecosystem fund fee ratio must be 0 <= ratio <= 1")
	}

	if !isPercent(market.ExchangeFeeRatio) {
		return fmt.Errorf("exchange fee ratio must be 0 <= ratio <= 1")
	}

	if !isPercent(market.LiquidationFeeRatio) {
		return fmt.Errorf("liquidation fee ratio must be 0 <= ratio <= 1")
	}

	if !isPercent(market.PartialLiquidationRatio) {
		return fmt.Errorf("partial liquidation ratio must be 0 <= ratio <= 1")
	}

	if market.MaxLeverage.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("max leverage must be > 0")
	}

	if sdk.OneDec().Quo(market.MaxLeverage).LT(market.MaintenanceMarginRatio) {
		return fmt.Errorf("margin ratio opened with max leverage position will be lower than Maintenance margin ratio")
	}

	return nil
}

/*
IsOverFluctuationLimitInRelationWithSnapshot compares the updated pool's spot price with the current spot price.

If the fluctuation limit ratio is zero, then the fluctuation limit check is skipped.

args:
  - pool: the updated market
  - snapshot: the snapshot to compare against

ret:
  - bool: true if the fluctuation limit is violated. false otherwise
*/
func (market Market) IsOverFluctuationLimitInRelationWithSnapshot(amm AMM, snapshot ReserveSnapshot) bool {
	if market.PriceFluctuationLimitRatio.IsZero() {
		return false
	}

	markPrice := amm.MarkPrice()
	snapshotUpperLimit := snapshot.upperLimit(market.PriceFluctuationLimitRatio)
	snapshotLowerLimit := snapshot.lowerLimit(market.PriceFluctuationLimitRatio)

	if markPrice.GT(snapshotUpperLimit) || markPrice.LT(snapshotLowerLimit) {
		return true
	}

	return false
}

func (market *Market) WithPriceFluctuationLimitRatio(value sdk.Dec) *Market {
	market.PriceFluctuationLimitRatio = value
	return market
}

func (market *Market) WithMaintenanceMarginRatio(value sdk.Dec) *Market {
	market.MaintenanceMarginRatio = value
	return market
}

func (market *Market) WithMaxLeverage(value sdk.Dec) *Market {
	market.MaxLeverage = value
	return market
}

func (market *Market) WithEcosystemFee(value sdk.Dec) *Market {
	market.EcosystemFundFeeRatio = value
	return market
}

func (market *Market) WithExchangeFee(value sdk.Dec) *Market {
	market.ExchangeFeeRatio = value
	return market
}

func (market *Market) WithLiquidationFee(value sdk.Dec) *Market {
	market.LiquidationFeeRatio = value
	return market
}

func (market *Market) WithPartialLiquidationRatio(value sdk.Dec) *Market {
	market.PartialLiquidationRatio = value
	return market
}

func (market *Market) WithFundingRateEpochId(value string) *Market {
	market.FundingRateEpochId = value
	return market
}

func (market *Market) WithPair(value asset.Pair) *Market {
	market.Pair = value
	return market
}

func (market *Market) WithLatestCumulativePremiumFraction(value sdk.Dec) *Market {
	market.LatestCumulativePremiumFraction = value
	return market
}

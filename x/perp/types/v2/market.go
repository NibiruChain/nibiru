package v2

import (
	fmt "fmt"

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

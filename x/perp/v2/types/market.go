package types

import (
	sdkmath "cosmossdk.io/math"
	fmt "fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func isPercent(v sdkmath.LegacyDec) bool {
	return v.GTE(sdkmath.LegacyZeroDec()) && v.LTE(sdkmath.LegacyOneDec())
}

func (market Market) Validate() error {
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

	if market.MaxLeverage.LTE(sdkmath.LegacyZeroDec()) {
		return fmt.Errorf("max leverage must be > 0")
	}

	if market.MaxFundingRate.LT(sdkmath.LegacyZeroDec()) {
		return fmt.Errorf("max funding rate must be >= 0")
	}

	if sdkmath.LegacyOneDec().Quo(market.MaxLeverage).LT(market.MaintenanceMarginRatio) {
		return fmt.Errorf("margin ratio opened with max leverage position will be lower than Maintenance margin ratio")
	}

	if err := market.OraclePair.Validate(); err != nil {
		return fmt.Errorf("err when validating oracle pair %w", err)
	}

	return nil
}

func (market Market) WithMaintenanceMarginRatio(value sdkmath.LegacyDec) Market {
	market.MaintenanceMarginRatio = value
	return market
}

func (market Market) WithMaxLeverage(value sdkmath.LegacyDec) Market {
	market.MaxLeverage = value
	return market
}

func (market Market) WithEcosystemFee(value sdkmath.LegacyDec) Market {
	market.EcosystemFundFeeRatio = value
	return market
}

func (market Market) WithExchangeFee(value sdkmath.LegacyDec) Market {
	market.ExchangeFeeRatio = value
	return market
}

func (market Market) WithLiquidationFee(value sdkmath.LegacyDec) Market {
	market.LiquidationFeeRatio = value
	return market
}

func (market Market) WithPartialLiquidationRatio(value sdkmath.LegacyDec) Market {
	market.PartialLiquidationRatio = value
	return market
}

func (market Market) WithFundingRateEpochId(value string) Market {
	market.FundingRateEpochId = value
	return market
}

func (market Market) WithMaxFundingRate(value sdkmath.LegacyDec) Market {
	market.MaxFundingRate = value
	return market
}

func (market Market) WithPair(value asset.Pair) Market {
	market.Pair = value
	return market
}

func (market Market) WithOraclePair(value asset.Pair) Market {
	market.OraclePair = value
	return market
}

func (market Market) WithLatestCumulativePremiumFraction(value sdkmath.LegacyDec) Market {
	market.LatestCumulativePremiumFraction = value
	return market
}

func (market Market) WithEcosystemFundFeeRatio(value sdkmath.LegacyDec) Market {
	market.EcosystemFundFeeRatio = value
	return market
}

func (market Market) WithExchangeFeeRatio(value sdkmath.LegacyDec) Market {
	market.ExchangeFeeRatio = value
	return market
}

func (market Market) WithLiquidationFeeRatio(value sdkmath.LegacyDec) Market {
	market.LiquidationFeeRatio = value
	return market
}

func (market Market) WithPrepaidBadDebt(value sdk.Coin) Market {
	market.PrepaidBadDebt = value
	return market
}

func (market Market) WithEnabled(value bool) Market {
	market.Enabled = value
	return market
}

func (market Market) WithTwapLookbackWindow(value time.Duration) Market {
	market.TwapLookbackWindow = value
	return market
}

func MarketsAreEqual(expected, actual Market) error {
	if expected.Pair != actual.Pair {
		return fmt.Errorf("expected market pair %s, got %s", expected.Pair, actual.Pair)
	}

	if !expected.MaintenanceMarginRatio.Equal(actual.MaintenanceMarginRatio) {
		return fmt.Errorf("expected market maintenance margin ratio %s, got %s", expected.MaintenanceMarginRatio, actual.MaintenanceMarginRatio)
	}

	if !expected.MaxLeverage.Equal(actual.MaxLeverage) {
		return fmt.Errorf("expected market max leverage %s, got %s", expected.MaxLeverage, actual.MaxLeverage)
	}

	if !expected.EcosystemFundFeeRatio.Equal(actual.EcosystemFundFeeRatio) {
		return fmt.Errorf("expected market ecosystem fund fee ratio %s, got %s", expected.EcosystemFundFeeRatio, actual.EcosystemFundFeeRatio)
	}

	if !expected.ExchangeFeeRatio.Equal(actual.ExchangeFeeRatio) {
		return fmt.Errorf("expected market exchange fee ratio %s, got %s", expected.ExchangeFeeRatio, actual.ExchangeFeeRatio)
	}

	if !expected.LiquidationFeeRatio.Equal(actual.LiquidationFeeRatio) {
		return fmt.Errorf("expected market liquidation fee ratio %s, got %s", expected.LiquidationFeeRatio, actual.LiquidationFeeRatio)
	}

	if !expected.PartialLiquidationRatio.Equal(actual.PartialLiquidationRatio) {
		return fmt.Errorf("expected market partial liquidation ratio %s, got %s", expected.PartialLiquidationRatio, actual.PartialLiquidationRatio)
	}

	if expected.FundingRateEpochId != actual.FundingRateEpochId {
		return fmt.Errorf("expected market funding rate epoch id %s, got %s", expected.FundingRateEpochId, actual.FundingRateEpochId)
	}

	if !expected.MaxFundingRate.Equal(actual.MaxFundingRate) {
		return fmt.Errorf("expected market max funding rate %s, got %s", expected.MaxFundingRate, actual.MaxFundingRate)
	}

	if !expected.PrepaidBadDebt.Equal(actual.PrepaidBadDebt) {
		return fmt.Errorf("expected market prepaid bad debt %s, got %s", expected.PrepaidBadDebt, actual.PrepaidBadDebt)
	}

	if expected.Enabled != actual.Enabled {
		return fmt.Errorf("expected market enabled %t, got %t", expected.Enabled, actual.Enabled)
	}

	if expected.TwapLookbackWindow != actual.TwapLookbackWindow {
		return fmt.Errorf("expected market twap lookback window %s, got %s", expected.TwapLookbackWindow, actual.TwapLookbackWindow)
	}

	if expected.OraclePair != actual.OraclePair {
		return fmt.Errorf("expected oracle pair %s, got %s", expected.OraclePair, actual.OraclePair)
	}

	if !expected.LatestCumulativePremiumFraction.Equal(actual.LatestCumulativePremiumFraction) {
		return fmt.Errorf(
			"expected market latest cumulative premium fraction %s, got %s",
			expected.LatestCumulativePremiumFraction,
			actual.LatestCumulativePremiumFraction,
		)
	}

	return nil
}

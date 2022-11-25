package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// HasEnoughQuoteReserve returns true if there is enough quote reserve based on
// quoteReserve * tradeLimitRatio
func (vpool *Vpool) HasEnoughQuoteReserve(quoteAmount sdk.Dec) bool {
	return vpool.QuoteAssetReserve.Mul(vpool.Config.TradeLimitRatio).GTE(quoteAmount)
}

// HasEnoughBaseReserve returns true if there is enough base reserve based on
// baseReserve * tradeLimitRatio
func (vpool *Vpool) HasEnoughBaseReserve(baseAmount sdk.Dec) bool {
	return vpool.BaseAssetReserve.Mul(vpool.Config.TradeLimitRatio).GTE(baseAmount)
}

/*
GetBaseAmountByQuoteAmount returns the amount of base asset you will get out
by giving a specified amount of quote asset

args:
  - dir: add to pool or remove from pool
  - quoteAmount: the amount of quote asset to add to/remove from the pool

ret:
  - baseAmountOut: the amount of base assets required to make this hypothetical swap
    always an absolute value
  - err: error
*/
func (vpool *Vpool) GetBaseAmountByQuoteAmount(
	dir Direction,
	quoteAmount sdk.Dec,
) (baseAmount sdk.Dec, err error) {
	if quoteAmount.IsZero() {
		return sdk.ZeroDec(), nil
	}

	invariant := vpool.QuoteAssetReserve.Mul(vpool.BaseAssetReserve) // x * y = k

	var quoteAssetsAfter sdk.Dec
	if dir == Direction_ADD_TO_POOL {
		quoteAssetsAfter = vpool.QuoteAssetReserve.Add(quoteAmount)
	} else {
		quoteAssetsAfter = vpool.QuoteAssetReserve.Sub(quoteAmount)
	}

	if quoteAssetsAfter.LTE(sdk.ZeroDec()) {
		return sdk.Dec{}, ErrQuoteReserveAtZero
	}

	baseAssetsAfter := invariant.Quo(quoteAssetsAfter)
	baseAmount = baseAssetsAfter.Sub(vpool.BaseAssetReserve).Abs()

	return baseAmount, nil
}

/*
GetQuoteAmountByBaseAmount returns the amount of quote asset you will get out
by giving a specified amount of base asset

args:
  - dir: add to pool or remove from pool
  - baseAmount: the amount of base asset to add to/remove from the pool

ret:
  - quoteAmountOut: the amount of quote assets required to make this hypothetical swap
    always an absolute value
  - err: error
*/
func (vpool *Vpool) GetQuoteAmountByBaseAmount(
	dir Direction, baseAmount sdk.Dec,
) (quoteAmount sdk.Dec, err error) {
	if baseAmount.IsZero() {
		return sdk.ZeroDec(), nil
	}

	invariant := vpool.QuoteAssetReserve.Mul(vpool.BaseAssetReserve) // x * y = k

	var baseAssetsAfter sdk.Dec
	if dir == Direction_ADD_TO_POOL {
		baseAssetsAfter = vpool.BaseAssetReserve.Add(baseAmount)
	} else {
		baseAssetsAfter = vpool.BaseAssetReserve.Sub(baseAmount)
	}

	if baseAssetsAfter.LTE(sdk.ZeroDec()) {
		return sdk.Dec{}, ErrBaseReserveAtZero.Wrapf(
			"base assets below zero after trying to swap %s base assets",
			baseAmount.String(),
		)
	}

	quoteAssetsAfter := invariant.Quo(baseAssetsAfter)
	quoteAmount = quoteAssetsAfter.Sub(vpool.QuoteAssetReserve).Abs()

	return quoteAmount, nil
}

// IncreaseBaseAssetReserve increases the quote reserve by amount
func (vpool *Vpool) IncreaseBaseAssetReserve(amount sdk.Dec) {
	vpool.BaseAssetReserve = vpool.BaseAssetReserve.Add(amount)
}

// DecreaseBaseAssetReserve descreases the quote asset reserve by amount
func (vpool *Vpool) DecreaseBaseAssetReserve(amount sdk.Dec) {
	vpool.BaseAssetReserve = vpool.BaseAssetReserve.Sub(amount)
}

func (vpool *Vpool) IncreaseQuoteAssetReserve(amount sdk.Dec) {
	vpool.QuoteAssetReserve = vpool.QuoteAssetReserve.Add(amount)
}

// DecreaseQuoteAssetReserve decreases the base reserve by amount
func (vpool *Vpool) DecreaseQuoteAssetReserve(amount sdk.Dec) {
	vpool.QuoteAssetReserve = vpool.QuoteAssetReserve.Sub(amount)
}

// ValidateReserves checks that reserves are positive.
func (vpool *Vpool) ValidateReserves() error {
	if !vpool.QuoteAssetReserve.IsPositive() || !vpool.BaseAssetReserve.IsPositive() {
		return ErrNonPositiveReserves.Wrap("pool: " + vpool.String())
	} else {
		return nil
	}
}

func (cfg *VpoolConfig) Validate() error {
	// trade limit ratio always between 0 and 1
	if cfg.TradeLimitRatio.LT(sdk.ZeroDec()) || cfg.TradeLimitRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("trade limit ratio must be 0 <= ratio <= 1")
	}

	// fluctuation limit ratio between 0 and 1
	if cfg.FluctuationLimitRatio.LT(sdk.ZeroDec()) || cfg.FluctuationLimitRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("fluctuation limit ratio must be 0 <= ratio <= 1")
	}

	// max oracle spread ratio between 0 and 1
	if cfg.MaxOracleSpreadRatio.LT(sdk.ZeroDec()) || cfg.MaxOracleSpreadRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("max oracle spread ratio must be 0 <= ratio <= 1")
	}

	if cfg.MaintenanceMarginRatio.LT(sdk.ZeroDec()) || cfg.MaintenanceMarginRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("maintenance margin ratio ratio must be 0 <= ratio <= 1")
	}

	if cfg.MaxLeverage.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("max leverage must be > 0")
	}

	if sdk.OneDec().Quo(cfg.MaxLeverage).LT(cfg.MaintenanceMarginRatio) {
		return fmt.Errorf("margin ratio opened with max leverage position will be lower than Maintenance margin ratio")
	}

	return nil
}

func (vpool *Vpool) Validate() error {
	if err := vpool.Pair.Validate(); err != nil {
		return fmt.Errorf("invalid asset pair: %w", err)
	}

	// base asset reserve always > 0
	// quote asset reserve always > 0
	if err := vpool.ValidateReserves(); err != nil {
		return err
	}

	if err := vpool.Config.Validate(); err != nil {
		return err
	}

	return nil
}

// GetMarkPrice returns the price of the asset.
func (vpool Vpool) GetMarkPrice() sdk.Dec {
	if vpool.BaseAssetReserve.IsNil() || vpool.BaseAssetReserve.IsZero() ||
		vpool.QuoteAssetReserve.IsNil() || vpool.QuoteAssetReserve.IsZero() {
		return sdk.ZeroDec()
	}

	return vpool.QuoteAssetReserve.Quo(vpool.BaseAssetReserve)
}

/*
IsOverFluctuationLimitInRelationWithSnapshot compares the updated pool's spot price with the current spot price.

If the fluctuation limit ratio is zero, then the fluctuation limit check is skipped.

args:
  - pool: the updated vpool
  - snapshot: the snapshot to compare against

ret:
  - bool: true if the fluctuation limit is violated. false otherwise
*/
func (vpool Vpool) IsOverFluctuationLimitInRelationWithSnapshot(snapshot ReserveSnapshot) bool {
	if vpool.Config.FluctuationLimitRatio.IsZero() {
		return false
	}

	markPrice := vpool.GetMarkPrice()
	snapshotUpperLimit := snapshot.GetUpperMarkPriceFluctuationLimit(
		vpool.Config.FluctuationLimitRatio)
	snapshotLowerLimit := snapshot.GetLowerMarkPriceFluctuationLimit(
		vpool.Config.FluctuationLimitRatio)

	if markPrice.GT(snapshotUpperLimit) || markPrice.LT(snapshotLowerLimit) {
		return true
	}

	return false
}

/*
IsOverSpreadLimit compares the current mark price of the vpool
to the underlying's index price.
It panics if you provide it with a pair that doesn't exist in the state.

args:
  - indexPrice: the index price we want to compare.

ret:
  - bool: whether or not the price has deviated from the oracle price beyond a spread ratio
*/
func (vpool Vpool) IsOverSpreadLimit(indexPrice sdk.Dec) bool {
	return vpool.GetMarkPrice().Sub(indexPrice).
		Quo(indexPrice).Abs().GTE(vpool.Config.MaxOracleSpreadRatio)
}

func (vpool Vpool) ToSnapshot(ctx sdk.Context) ReserveSnapshot {
	snapshot := NewReserveSnapshot(
		vpool.Pair,
		vpool.BaseAssetReserve,
		vpool.QuoteAssetReserve,
		ctx.BlockTime(),
	)
	if err := snapshot.Validate(); err != nil {
		panic(err)
	}
	return snapshot
}

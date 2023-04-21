package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
)

// ----------------------------------------------------------------------------
// Market - core functions
// ----------------------------------------------------------------------------

/*
FromQuoteAssetToReserve returns the amount of quote reserve equivalent to the
amount of quote asset given
*/
func (market *Market) FromQuoteAssetToReserve(quoteAsset sdk.Dec) sdk.Dec {
	return quoteAsset.Quo(market.PegMultiplier)
}

/*
FromQuoteReserveToAsset returns the amount of quote asset equivalent to the
amount of quote reserve given
*/
func (market *Market) FromQuoteReserveToAsset(quoteReserve sdk.Dec) sdk.Dec {
	return quoteReserve.Mul(market.PegMultiplier)
}

/*
GetBaseAmountByQuoteAmount returns the amount of base asset you will get out
by giving a specified amount of quote asset

args:
  - quoteDelta: the amount of quote asset to add to/remove from the pool.
    Adding to the quote reserves is synonymous with positive 'quoteDelta'.

ret:
  - baseOutAbs: the amount of base assets required to make this hypothetical swap
    always an absolute value
  - err: error
*/
func (market *Market) GetBaseAmountByQuoteAmount(
	quoteDelta sdk.Dec,
) (baseOutAbs sdk.Dec, err error) {
	if quoteDelta.IsZero() {
		return sdk.ZeroDec(), nil
	}

	invariant := market.QuoteReserve.Mul(market.BaseReserve) // x * y = k

	quoteReservesAfter := market.QuoteReserve.Add(quoteDelta)
	if quoteReservesAfter.LTE(sdk.ZeroDec()) {
		return sdk.Dec{}, ErrQuoteReserveAtZero
	}

	baseReservesAfter := invariant.Quo(quoteReservesAfter)
	baseOutAbs = baseReservesAfter.Sub(market.BaseReserve).Abs()

	return baseOutAbs, nil
}

/*
GetQuoteReserveByBase returns the amount of quote asset you will get out
by giving a specified amount of base asset

args:
  - dir: add to pool or remove from pool
  - baseAmount: the amount of base asset to add to/remove from the pool

ret:
  - quoteOutAbs: the amount of quote assets required to make this hypothetical swap
    always an absolute value
  - err: error
*/
func (market *Market) GetQuoteReserveByBase(
	baseDelta sdk.Dec,
) (quoteOutAbs sdk.Dec, err error) {
	if baseDelta.IsZero() {
		return sdk.ZeroDec(), nil
	}

	invariant := market.QuoteReserve.Mul(market.BaseReserve) // x * y = k

	baseReservesAfter := market.BaseReserve.Add(baseDelta)
	if baseReservesAfter.LTE(sdk.ZeroDec()) {
		return sdk.Dec{}, ErrBaseReserveAtZero.Wrapf(
			"base assets below zero after trying to swap %s base assets",
			baseDelta.String(),
		)
	}

	quoteReservesAfter := invariant.Quo(baseReservesAfter)
	quoteOutAbs = quoteReservesAfter.Sub(market.QuoteReserve).Abs()

	return quoteOutAbs, nil
}

// GetMarkPrice returns the price of the asset.
func (market Market) GetMarkPrice() sdk.Dec {
	if market.BaseReserve.IsNil() || market.BaseReserve.IsZero() ||
		market.QuoteReserve.IsNil() || market.QuoteReserve.IsZero() {
		return sdk.ZeroDec()
	}

	return market.QuoteReserve.Quo(market.BaseReserve).Mul(market.PegMultiplier)
}

// AddToQuoteReserve adds 'amount' to the quote asset reserves
// The 'amount' is not assumed to be positive.
func (market *Market) AddToQuoteReserve(amount sdk.Dec) {
	market.QuoteReserve = market.QuoteReserve.Add(amount)
}

// AddToBaseReserveAndBias adds 'amount' to the base asset reserves
// The 'amount' is not assumed to be positive.
func (market *Market) AddToBaseReserveAndBias(amount sdk.Dec) {
	market.BaseReserve = market.BaseReserve.Add(amount)
	market.Bias = market.Bias.Sub(amount) // Bias is the opposite of what the trader get
}

type ArgsNewMarket struct {
	Pair          asset.Pair
	BaseReserves  sdk.Dec
	QuoteReserves sdk.Dec
	Config        *MarketConfig
	Bias          sdk.Dec
	PegMultiplier sdk.Dec
}

func NewMarket(args ArgsNewMarket) Market {
	var config MarketConfig
	if args.Config != nil {
		config = *args.Config
	} else {
		config = *DefaultMarketConfig()
	}

	return Market{
		Pair:          args.Pair,
		BaseReserve:   args.BaseReserves,
		QuoteReserve:  args.QuoteReserves,
		Config:        config,
		SqrtDepth:     common.MustSqrtDec(args.QuoteReserves.Mul(args.BaseReserves)),
		Bias:          args.Bias,
		PegMultiplier: args.PegMultiplier,
	}
}

func (market *Market) ComputeSqrtDepth() (sqrtDepth sdk.Dec, err error) {
	liqDepth := market.QuoteReserve.Mul(market.BaseReserve)
	return common.SqrtDec(liqDepth)
}

func (market *Market) InitLiqDepth() (Market, error) {
	sqrtDepth, err := market.ComputeSqrtDepth()
	if err != nil {
		return Market{}, err
	}

	pool := *market
	pool.SqrtDepth = sqrtDepth
	return pool, nil
}

// String returns the string representation of the pool. Note that this differs
// from the default output of the proto-generated 'String' method.
func (pool *Market) String() string {
	elems := []string{
		fmt.Sprintf("pair: %s", pool.Pair),
		fmt.Sprintf("base_reserves: %s", pool.BaseReserve),
		fmt.Sprintf("quote_reserves: %s", pool.QuoteReserve),
		fmt.Sprintf("sqrt_depth: %s", pool.SqrtDepth),
		fmt.Sprintf("config: %s", &pool.Config),
	}
	elemString := strings.Join(elems, ", ")
	return "{ " + elemString + " }"
}

// ----------------------------------------------------------------------------
// MarketConfig
// ----------------------------------------------------------------------------

func (cfg *MarketConfig) Validate() error {
	// trade limit ratio always between 0 and 1
	if cfg.TradeLimitRatio.LT(sdk.ZeroDec()) || cfg.TradeLimitRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("trade limit ratio of must be 0 <= ratio <= 1, not %s",
			cfg.TradeLimitRatio)
	}

	// fluctuation limit ratio between 0 and 1
	if cfg.FluctuationLimitRatio.LT(sdk.ZeroDec()) || cfg.FluctuationLimitRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("fluctuation limit ratio must be 0 <= ratio <= 1, not %s",
			cfg.FluctuationLimitRatio)
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

func DefaultMarketConfig() *MarketConfig {
	return &MarketConfig{
		TradeLimitRatio:        sdk.MustNewDecFromStr("0.1"),
		FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
		MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
		MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
		// 0.0625 = 1 / 16. This implies that an effective leverage of 16x is
		// what defines the liquidation threshold and maintenance margin ratio.
		MaxLeverage: sdk.NewDec(10),
	}
}

func (poolCfg *MarketConfig) SetConfig(cfg MarketConfig) *MarketConfig {
	poolCfg.TradeLimitRatio = cfg.TradeLimitRatio
	poolCfg.FluctuationLimitRatio = cfg.FluctuationLimitRatio
	poolCfg.MaxOracleSpreadRatio = cfg.MaxOracleSpreadRatio
	poolCfg.MaintenanceMarginRatio = cfg.MaintenanceMarginRatio
	poolCfg.MaxLeverage = cfg.MaxLeverage
	return poolCfg
}

func (poolCfg *MarketConfig) WithTradeLimitRatio(value sdk.Dec) *MarketConfig {
	newPoolCfg := new(MarketConfig).SetConfig(*poolCfg)
	newPoolCfg.TradeLimitRatio = value
	return newPoolCfg
}

func (poolCfg *MarketConfig) WithFluctuationLimitRatio(value sdk.Dec) *MarketConfig {
	newPoolCfg := new(MarketConfig).SetConfig(*poolCfg)
	newPoolCfg.FluctuationLimitRatio = value
	return newPoolCfg
}

func (poolCfg *MarketConfig) WithMaxOracleSpreadRatio(value sdk.Dec) *MarketConfig {
	newPoolCfg := new(MarketConfig).SetConfig(*poolCfg)
	newPoolCfg.MaxOracleSpreadRatio = value
	return newPoolCfg
}

func (poolCfg *MarketConfig) WithMaintenanceMarginRatio(value sdk.Dec) *MarketConfig {
	newPoolCfg := new(MarketConfig).SetConfig(*poolCfg)
	newPoolCfg.MaintenanceMarginRatio = value
	return newPoolCfg
}

func (poolCfg *MarketConfig) WithMaxLeverage(value sdk.Dec) *MarketConfig {
	newPoolCfg := new(MarketConfig).SetConfig(*poolCfg)
	newPoolCfg.MaxLeverage = value
	return newPoolCfg
}

// ----------------------------------------------------------------------------
// Market - validation functions
// ----------------------------------------------------------------------------

func (market *Market) Validate() error {
	if err := market.Pair.Validate(); err != nil {
		return fmt.Errorf("invalid asset pair: %w", err)
	}

	// base asset reserve always > 0
	// quote asset reserve always > 0
	if err := market.ValidateReserves(); err != nil {
		return err
	}
	if err := market.ValidateLiquidityDepth(); err != nil {
		return err
	}

	if market.PegMultiplier.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("peg multiplier must be > 0")
	}

	if err := market.Config.Validate(); err != nil {
		return err
	}

	return nil
}

// HasEnoughQuoteReserve returns true if there is enough quote reserve based on
// quoteReserve * tradeLimitRatio
func (market *Market) HasEnoughQuoteReserve(quoteAmount sdk.Dec) bool {
	return market.QuoteReserve.Mul(market.Config.TradeLimitRatio).GTE(quoteAmount.Abs())
}

// HasEnoughBaseReserve returns true if there is enough base reserve based on
// baseReserve * tradeLimitRatio
func (market *Market) HasEnoughBaseReserve(baseAmount sdk.Dec) bool {
	return market.BaseReserve.Mul(market.Config.TradeLimitRatio).GTE(baseAmount.Abs())
}

func (market *Market) HasEnoughReservesForTrade(
	quoteAmtAbs sdk.Dec, baseAmtAbs sdk.Dec,
) (err error) {
	if !market.HasEnoughQuoteReserve(quoteAmtAbs) {
		return ErrOverTradingLimit.Wrapf(
			"quote amount %s is over trading limit", quoteAmtAbs)
	}
	if !market.HasEnoughBaseReserve(baseAmtAbs) {
		return ErrOverTradingLimit.Wrapf(
			"base amount %s is over trading limit", baseAmtAbs)
	}

	return nil
}

// ValidateReserves checks that reserves are positive.
func (market *Market) ValidateReserves() error {
	if !market.QuoteReserve.IsPositive() || !market.BaseReserve.IsPositive() {
		return ErrNonPositiveReserves.Wrap("pool: " + market.String())
	} else {
		return nil
	}
}

// ValidateLiquidityDepth checks that reserves are positive.
func (market *Market) ValidateLiquidityDepth() error {
	computedSqrtDepth, err := market.ComputeSqrtDepth()
	if err != nil {
		return err
	}

	if !market.SqrtDepth.IsPositive() {
		return ErrLiquidityDepth.Wrap(
			"liq depth must be positive. pool: " + market.String())
	} else if !market.SqrtDepth.Sub(computedSqrtDepth).Abs().LTE(sdk.NewDec(1)) {
		return ErrLiquidityDepth.Wrap(
			"computed sqrt and current sqrt are mismatched. pool: " + market.String())
	} else {
		return nil
	}
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
func (market Market) IsOverFluctuationLimitInRelationWithSnapshot(snapshot ReserveSnapshot) bool {
	if market.Config.FluctuationLimitRatio.IsZero() {
		return false
	}

	markPrice := market.GetMarkPrice()
	snapshotUpperLimit := snapshot.GetUpperMarkPriceFluctuationLimit(
		market.Config.FluctuationLimitRatio)
	snapshotLowerLimit := snapshot.GetLowerMarkPriceFluctuationLimit(
		market.Config.FluctuationLimitRatio)

	if markPrice.GT(snapshotUpperLimit) || markPrice.LT(snapshotLowerLimit) {
		return true
	}

	return false
}

/*
IsOverSpreadLimit compares the current mark price of the market
to the underlying's index price.
It panics if you provide it with a pair that doesn't exist in the state.

args:
  - indexPrice: the index price we want to compare.

ret:
  - bool: whether or not the price has deviated from the oracle price beyond a spread ratio
*/
func (market Market) IsOverSpreadLimit(indexPrice sdk.Dec) bool {
	return market.GetMarkPrice().Sub(indexPrice).
		Quo(indexPrice).Abs().GTE(market.Config.MaxOracleSpreadRatio)
}

func (market Market) ToSnapshot(ctx sdk.Context) ReserveSnapshot {
	fmt.Println("Create shnaps")
	fmt.Println(market.String())
	snapshot := NewReserveSnapshot(
		market.Pair,
		market.BaseReserve,
		market.QuoteReserve,
		market.PegMultiplier,
		market.Bias,
		ctx.BlockTime(),
	)
	if err := snapshot.Validate(); err != nil {
		panic(err)
	}
	return snapshot
}

func (dir Direction) ToMultiplier() int64 {
	var dirMult int64
	switch dir {
	case Direction_LONG, Direction_DIRECTION_UNSPECIFIED:
		dirMult = 1
	case Direction_SHORT:
		dirMult = -1
	}
	return dirMult
}

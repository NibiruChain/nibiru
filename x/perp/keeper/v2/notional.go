package keeper

import (
	"fmt"
	"time"

	"github.com/NibiruChain/nibiru/x/perp/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PositionNotionalSpot returns the position's notional value based on the spot price.
func PositionNotionalSpot(amm v2types.AMM, position v2types.Position) (positionNotional sdk.Dec, err error) {
	// we want to know the price if the user closes their position
	// e.g. if the user has positive size, we want to short
	var dir v2types.Direction
	if position.Size_.IsPositive() {
		dir = v2types.Direction_SHORT
	} else {
		dir = v2types.Direction_LONG
	}

	quoteReserve, err := amm.GetQuoteReserveAmt(position.Size_.Abs(), dir)
	if err != nil {
		return sdk.Dec{}, err
	}
	return amm.FromQuoteReserveToAsset(quoteReserve), nil
}

// PositionNotionalTWAP returns the position's notional value based on the TWAP price.
func (k Keeper) PositionNotionalTWAP(ctx sdk.Context,
	position v2types.Position,
	twapLookbackWindow time.Duration,
) (positionNotional sdk.Dec, err error) {
	// we want to know the price if the user closes their position
	// e.g. if the user has positive size, we want to short
	var dir v2types.Direction
	if position.Size_.IsPositive() {
		dir = v2types.Direction_SHORT
	} else {
		dir = v2types.Direction_LONG
	}

	return k.BaseAssetTWAP(
		ctx,
		position.Pair,
		dir,
		position.Size_.Abs(),
		/*lookbackInterval=*/ twapLookbackWindow,
	)
}

// PositionNotionalOracle returns the position's notional value based on the oracle price.
func (k Keeper) PositionNotionalOracle(
	ctx sdk.Context,
	position v2types.Position,
) (positionNotional sdk.Dec, err error) {
	oraclePrice, err := k.OracleKeeper.GetExchangeRate(ctx, position.Pair)
	if err != nil {
		return sdk.Dec{}, err
	}
	return oraclePrice.Mul(position.Size_.Abs()), nil
}

// UnrealizedPnl calculates the unrealized profits and losses (PnL) of a position.
func UnrealizedPnl(position v2types.Position, positionNotional sdk.Dec) (unrealizedPnlSigned sdk.Dec) {
	if position.Size_.IsPositive() {
		// LONG
		return positionNotional.Sub(position.OpenNotional)
	} else {
		// SHORT
		return position.OpenNotional.Sub(positionNotional)
	}
}

/*
GetPreferencePositionNotionalAndUnrealizedPnL Calculates both position notional value and unrealized PnL based on
both spot price and TWAP, and lets the caller pick which one based on MAX or MIN.

args:
  - ctx: cosmos-sdk context
  - position: the trader's position
  - pnlPreferenceOption: MAX or MIN

Returns:
  - positionNotional: the position's notional value as sdk.Dec (signed)
  - unrealizedPnl: the position's unrealized profits and losses (PnL) as sdk.Dec (signed)
    For LONG positions, this is positionNotional - openNotional
    For SHORT positions, this is openNotional - positionNotional
*/
func (k Keeper) GetPreferencePositionNotionalAndUnrealizedPnL(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	position v2types.Position,
	pnLPreferenceOption types.PnLPreferenceOption,
) (positionNotional sdk.Dec, unrealizedPnl sdk.Dec, err error) {
	spotPositionNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return sdk.Dec{}, sdk.Dec{}, err
	}
	spotPricePnl := UnrealizedPnl(position, spotPositionNotional)

	twapPositionNotional, err := k.PositionNotionalTWAP(ctx, position, market.TwapLookbackWindow)
	if err != nil {
		return sdk.Dec{}, sdk.Dec{}, err
	}
	twapPnl := UnrealizedPnl(position, twapPositionNotional)

	switch pnLPreferenceOption {
	case types.PnLPreferenceOption_MAX:
		positionNotional = sdk.MaxDec(spotPositionNotional, twapPositionNotional)
		unrealizedPnl = sdk.MaxDec(spotPricePnl, twapPnl)
	case types.PnLPreferenceOption_MIN:
		positionNotional = sdk.MinDec(spotPositionNotional, twapPositionNotional)
		unrealizedPnl = sdk.MinDec(spotPricePnl, twapPnl)
	default:
		return sdk.Dec{}, sdk.Dec{}, fmt.Errorf(
			"invalid pnl preference option: %s", pnLPreferenceOption)
	}

	return positionNotional, unrealizedPnl, nil
}

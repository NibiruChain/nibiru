package keeper

import (
	"time"

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

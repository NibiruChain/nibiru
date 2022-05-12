package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SettlePosition settles a trader position
// TODO(mercilex): test
func (k Keeper) SettlePosition(ctx sdk.Context, pair common.TokenPair, trader string) (transferredCoins sdk.Coins, err error) {
	// TODO(mercilex): requireAmm?

	pos, err := k.Positions().Get(ctx, pair, trader)
	if err != nil {
		return
	}

	// check if pos size != 0
	if pos.Size_.IsZero() {
		return transferredCoins, types.ErrPositionSizeZero
	}
	// clear pos
	err = k.ClearPosition(ctx, pair, trader)
	if err != nil {
		return
	}

	// run calculations on settled values
	settlementPrice, err := k.VpoolKeeper.GetSettlementPrice(ctx, pair)
	if err != nil {
		return
	}
	settledValue := sdk.ZeroDec()

	switch settlementPrice.IsZero() {
	case true:
		settledValue = pos.Margin
	default:
		// openPrice = positionOpenNotional / abs(positionSize)
		openPrice := pos.OpenNotional.Quo(pos.Size_.Abs())
		// returnedFund := positionSize * (settlementPrice - openPrice) + positionMargin
		returnedFund := pos.Size_.Mul(settlementPrice.Sub(openPrice)).Add(pos.Margin)
		if returnedFund.GT(sdk.ZeroDec()) {
			settledValue = returnedFund.Abs()
		}
	}

	traderAddr, err := sdk.AccAddressFromBech32(trader)
	if err != nil {
		panic(err) // NOTE(mercilex): must never happen
	}

	transferredCoins = sdk.NewCoins() // TODO(mercilex): maybe here it would be cleaner to create a zero coin amount of the quote asset of the virtual pool
	if settledValue.GT(sdk.ZeroDec()) {
		// transfer, NOTE(mercilex): transferredCoins is over-written here.
		transferredCoins, err = k.Transfer(ctx, pair, traderAddr, settledValue.RoundInt())
		if err != nil {
			return
		}
	}

	events.EmitPositionSettle(ctx, pair.String(), traderAddr, transferredCoins)
	return
}

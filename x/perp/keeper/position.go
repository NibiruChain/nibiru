package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// SettlePosition settles a trader position
func (k Keeper) SettlePosition(
	ctx sdk.Context,
	currentPosition types.Position,
) (transferredCoins sdk.Coins, err error) {
	tokenPair, err := common.NewTokenPairFromStr(currentPosition.Pair)
	if err != nil {
		return sdk.Coins{}, err
	}

	if currentPosition.Size_.IsZero() {
		return sdk.NewCoins(), nil
	}

	err = k.ClearPosition(
		ctx,
		tokenPair,
		currentPosition.Address,
	)
	if err != nil {
		return
	}

	// run calculations on settled values
	settlementPrice, err := k.VpoolKeeper.GetSettlementPrice(ctx, tokenPair)
	if err != nil {
		return
	}

	settledValue := sdk.ZeroDec()
	if settlementPrice.IsZero() {
		settledValue = currentPosition.Margin
	} else {
		// openPrice = positionOpenNotional / abs(positionSize)
		openPrice := currentPosition.OpenNotional.Quo(currentPosition.Size_.Abs())
		// returnedFund := positionSize * (settlementPrice - openPrice) + positionMargin
		returnedFund := currentPosition.Size_.Mul(
			settlementPrice.Sub(openPrice)).Add(currentPosition.Margin)
		if returnedFund.IsPositive() {
			settledValue = returnedFund
		}
	}

	transferredCoins = sdk.NewCoins() // TODO(mercilex): maybe here it would be cleaner to create a zero coin amount of the quote asset of the virtual pool
	if settledValue.IsPositive() {
		transferredCoins, err = k.Transfer(
			ctx,
			tokenPair.GetQuoteTokenDenom(),
			sdk.AccAddress(currentPosition.Address),
			settledValue.RoundInt(),
		)
		if err != nil {
			return sdk.Coins{}, err
		}
	}

	events.EmitPositionSettle(
		ctx,
		tokenPair.String(),
		currentPosition.Address,
		transferredCoins,
	)

	return
}

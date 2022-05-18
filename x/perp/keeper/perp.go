package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// TODO test: ClearPosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) ClearPosition(ctx sdk.Context, pair common.TokenPair, trader string) error {
	return k.Positions().Update(ctx, &types.Position{
		Address:                             trader,
		Pair:                                pair.String(),
		Size_:                               sdk.ZeroDec(),
		Margin:                              sdk.ZeroDec(),
		OpenNotional:                        sdk.ZeroDec(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
		LiquidityHistoryIndex:               0,
		BlockNumber:                         ctx.BlockHeight(),
	})
}

func (k Keeper) GetPosition(
	ctx sdk.Context, pair common.TokenPair, owner string,
) (*types.Position, error) {
	return k.Positions().Get(ctx, pair, owner)
}

func (k Keeper) SetPosition(
	ctx sdk.Context, pair common.TokenPair, owner string,
	position *types.Position) {
	k.Positions().Set(ctx, pair, owner, position)
}

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

	transferredCoins = sdk.NewCoins(sdk.NewInt64Coin(tokenPair.GetQuoteTokenDenom(), 0))
	if settledValue.IsPositive() {
		toTransfer := sdk.NewCoin(tokenPair.GetQuoteTokenDenom(), settledValue.RoundInt())
		transferredCoins = sdk.NewCoins(toTransfer)
		addr, err := sdk.AccAddressFromBech32(currentPosition.Address)
		if err != nil {
			panic(err) // NOTE(mercilex): must never happen
		}
		err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.VaultModuleAccount, addr, transferredCoins)
		if err != nil {
			panic(err) // NOTE(mercilex): must never happen
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

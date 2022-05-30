package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// TODO test: ClearPosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) ClearPosition(ctx sdk.Context, pair common.AssetPair, traderAddr sdk.AccAddress) error {
	return k.Positions().Update(ctx, &types.Position{
		TraderAddress:                       traderAddr,
		Pair:                                pair.String(),
		Size_:                               sdk.ZeroDec(),
		Margin:                              sdk.ZeroDec(),
		OpenNotional:                        sdk.ZeroDec(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
		BlockNumber:                         ctx.BlockHeight(),
	})
}

func (k Keeper) GetPosition(
	ctx sdk.Context, pair common.AssetPair, traderAddr sdk.AccAddress,
) (*types.Position, error) {
	return k.Positions().Get(ctx, pair, traderAddr)
}

func (k Keeper) SetPosition(
	ctx sdk.Context, pair common.AssetPair, traderAddr sdk.AccAddress,
	position *types.Position) {
	k.Positions().Set(ctx, pair, traderAddr, position)
}

// SettlePosition settles a trader position
func (k Keeper) SettlePosition(
	ctx sdk.Context,
	currentPosition types.Position,
) (transferredCoins sdk.Coins, err error) {
	tokenPair, err := common.NewAssetPairFromStr(currentPosition.Pair)
	if err != nil {
		return sdk.Coins{}, err
	}

	if currentPosition.Size_.IsZero() {
		return sdk.NewCoins(), nil
	}

	err = k.ClearPosition(
		ctx,
		tokenPair,
		currentPosition.TraderAddress,
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
	settledValueInt := settledValue.RoundInt()
	if settledValueInt.IsPositive() {
		toTransfer := sdk.NewCoin(tokenPair.GetQuoteTokenDenom(), settledValueInt)
		transferredCoins = sdk.NewCoins(toTransfer)
		if err != nil {
			panic(err) // NOTE(mercilex): must never happen
		}
		err = k.BankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.VaultModuleAccount,
			currentPosition.TraderAddress,
			transferredCoins,
		)
		if err != nil {
			panic(err) // NOTE(mercilex): must never happen
		}
	}

	events.EmitPositionSettle(
		ctx,
		tokenPair.String(),
		currentPosition.TraderAddress.String(),
		transferredCoins,
	)

	return
}

func (k Keeper) ClosePosition(ctx sdk.Context, pair common.AssetPair, addr sdk.AccAddress) (*types.PositionResp, error) {
	// check that we're not paused
	params := k.GetParams(ctx)
	if params.Stopped {
		return nil, types.ErrExchangeStopped
	}

	if err := k.requireVpool(ctx, pair); err != nil {
		return nil, err
	}

	// TODO(mercilex): require not restriction mode??
	position, err := k.GetPosition(ctx, pair, addr)
	if err != nil {
		return nil, err
	}
	if position.Size_.IsZero() {
		return nil, types.ErrPositionZero
	}

	resp, err := k.closePositionEntirely(ctx, *position, sdk.ZeroDec())
	if err != nil {
		return nil, err
	}

	// check if there's bad debt
	if !resp.BadDebt.IsZero() {
		panic("bad debt")
	}

	k.Withdraw()

	return resp, nil
}

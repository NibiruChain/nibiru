package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

/* AddMargin deleverages an existing position by adding margin (collateral)
to it. Adding margin increases the margin ratio of the corresponding position.
*/
func (k Keeper) AddMargin(
	goCtx context.Context, msg *types.MsgAddMargin,
) (res *types.MsgAddMarginResponse, err error) {
	// ------------- Message Setup -------------

	ctx := sdk.UnwrapSDKContext(goCtx)

	// validate margin amount
	addedMargin := msg.Margin.Amount
	if !addedMargin.IsPositive() {
		return res, fmt.Errorf("margin must be positive, not: %v", addedMargin.String())
	}

	// validate pair
	pair, err := common.NewTokenPairFromStr(msg.TokenPair)
	if err != nil {
		return res, err
	}
	err = k.requireVpool(ctx, pair)
	if err != nil {
		return res, err
	}

	// validate margin denom
	if msg.Margin.Denom != pair.GetQuoteTokenDenom() {
		return res, fmt.Errorf("invalid margin denom")
	}

	// ------------- AddMargin -------------

	position, err := k.Positions().Get(ctx, pair, msg.Sender)
	if err != nil {
		return res, err
	}

	position.Margin = position.Margin.Add(addedMargin.ToDec())

	coinToSend := sdk.NewCoin(pair.GetQuoteTokenDenom(), addedMargin)
	vaultAddr := k.AccountKeeper.GetModuleAddress(types.VaultModuleAccount)
	if err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx, msg.Sender, types.VaultModuleAccount, sdk.NewCoins(coinToSend),
	); err != nil {
		return res, err
	}
	events.EmitTransfer(ctx,
		/* coin */ coinToSend,
		/* from */ vaultAddr,
		/* to */ msg.Sender,
	)

	k.Positions().Set(ctx, pair, msg.Sender, position)

	fPayment := sdk.ZeroDec()
	events.EmitMarginChange(ctx, msg.Sender, pair.String(), addedMargin, fPayment)
	return &types.MsgAddMarginResponse{}, nil
}

/* RemoveMargin further leverages an existing position by directly removing
the margin (collateral) that backs it from the vault. This also decreases the
margin ratio of the position.
*/
func (k Keeper) RemoveMargin(
	goCtx context.Context, msg *types.MsgRemoveMargin,
) (res *types.MsgRemoveMarginResponse, err error) {
	// ------------- Message Setup -------------

	ctx := sdk.UnwrapSDKContext(goCtx)

	// validate trader

	if err = sdk.VerifyAddressFormat(msg.Sender); err != nil {
		return nil, err
	}

	// validate margin amount
	margin := msg.Margin.Amount
	if margin.LTE(sdk.ZeroInt()) {
		return res, fmt.Errorf("margin must be positive, not: %v", margin.String())
	}

	// validate pair
	pair, err := common.NewTokenPairFromStr(msg.TokenPair)
	if err != nil {
		return res, err
	}
	err = k.requireVpool(ctx, pair)
	if err != nil {
		return res, err
	}

	// validate margin denom
	if msg.Margin.Denom != pair.GetQuoteTokenDenom() {
		return res, fmt.Errorf("invalid margin denom")
	}

	// ------------- RemoveMargin -------------

	position, err := k.Positions().Get(ctx, pair, msg.Sender)
	if err != nil {
		return res, err
	}

	marginDelta := margin.Neg()
	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx, *position, marginDelta.ToDec())
	if err != nil {
		return res, err
	}
	position.Margin = remaining.Margin
	position.LastUpdateCumulativePremiumFraction = remaining.LatestCumulativePremiumFraction
	if !remaining.BadDebt.IsZero() {
		return res, fmt.Errorf("failed to remove margin; position has bad debt")
	}

	freeCollateral, err := k.calcFreeCollateral(
		ctx, *position, remaining.FundingPayment)
	if err != nil {
		return res, err
	} else if !freeCollateral.IsPositive() {
		return res, fmt.Errorf("not enough free collateral")
	}

	k.Positions().Set(ctx, pair, msg.Sender, position)

	coinToSend := sdk.NewCoin(pair.GetQuoteTokenDenom(), margin)
	err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.VaultModuleAccount, msg.Sender, sdk.NewCoins(coinToSend))
	if err != nil {
		return res, err
	}
	vaultAddr := k.AccountKeeper.GetModuleAddress(types.VaultModuleAccount)

	events.EmitTransfer(ctx,
		/* coin */ coinToSend,
		/* from */ vaultAddr,
		/* to */ msg.Sender,
	)

	events.EmitMarginChange(ctx, msg.Sender, pair.String(), margin, remaining.FundingPayment)
	return &types.MsgRemoveMarginResponse{
		MarginOut:      coinToSend,
		FundingPayment: remaining.FundingPayment,
	}, nil
}

// GetMarginRatio calculates the MarginRatio from a Position
func (k Keeper) GetMarginRatio(
	ctx sdk.Context, position types.Position, priceOption types.MarginCalculationPriceOption,
) (marginRatio sdk.Dec, err error) {
	if position.Size_.IsZero() {
		return sdk.Dec{}, types.ErrPositionZero
	}

	var (
		unrealizedPnL    sdk.Dec
		positionNotional sdk.Dec
	)

	switch priceOption {
	case types.MarginCalculationPriceOption_MAX_PNL:
		positionNotional, unrealizedPnL, err = k.getPreferencePositionNotionalAndUnrealizedPnL(
			ctx,
			position,
			types.PnLPreferenceOption_MAX,
		)
	case types.MarginCalculationPriceOption_INDEX:
		positionNotional, unrealizedPnL, err = k.getPositionNotionalAndUnrealizedPnL(
			ctx,
			position,
			types.PnLCalcOption_ORACLE,
		)
	case types.MarginCalculationPriceOption_SPOT:
		positionNotional, unrealizedPnL, err = k.getPositionNotionalAndUnrealizedPnL(
			ctx,
			position,
			types.PnLCalcOption_SPOT_PRICE,
		)
	}

	if err != nil {
		return sdk.Dec{}, err
	}
	if positionNotional.IsZero() {
		// NOTE causes division by zero in margin ratio calculation
		return sdk.Dec{},
			fmt.Errorf("margin ratio doesn't make sense with zero position notional")
	}

	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx,
		/* oldPosition */ position,
		/* marginDelta */ unrealizedPnL,
	)
	if err != nil {
		return sdk.Dec{}, err
	}

	marginRatio = remaining.Margin.Sub(remaining.BadDebt).
		Quo(positionNotional)
	return marginRatio, nil
}

func (k *Keeper) requireVpool(ctx sdk.Context, pair common.TokenPair) error {
	if !k.VpoolKeeper.ExistsPool(ctx, pair) {
		return fmt.Errorf("%v: %v", types.ErrPairNotFound.Error(), pair.String())
	}
	return nil
}

/*
requireMoreMarginRatio checks if the marginRatio corresponding to the margin
backing a position is above or below the 'baseMarginRatio'.
If 'largerThanOrEqualTo' is true, 'marginRatio' must be >= 'baseMarginRatio'.

Args:
  marginRatio: Ratio of the value of the margin and corresponding position(s).
    marginRatio is defined as (margin + unrealizedPnL) / notional
  baseMarginRatio: Specifies the threshold value that 'marginRatio' must meet.
  largerThanOrEqualTo: Specifies whether 'marginRatio' should be larger or
    smaller than 'baseMarginRatio'.
*/
func requireMoreMarginRatio(marginRatio, baseMarginRatio sdk.Dec, largerThanOrEqualTo bool) error {
	if largerThanOrEqualTo {
		if !marginRatio.GTE(baseMarginRatio) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	} else {
		if !marginRatio.LT(baseMarginRatio) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	}
	return nil
}

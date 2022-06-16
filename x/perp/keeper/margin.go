package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
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

	// validate trader
	msgSender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// validate margin amount
	if !msg.Margin.Amount.IsPositive() {
		err = fmt.Errorf("margin must be positive, not: %v", msg.Margin.Amount.String())
		k.Logger(ctx).Debug(
			err.Error(),
			"margin_amount",
			msg.Margin.Amount.String(),
		)
		return nil, err
	}

	// validate token pair
	pair, err := common.NewAssetPairFromStr(msg.TokenPair)
	if err != nil {
		k.Logger(ctx).Debug(
			err.Error(),
			"token_pair",
			msg.TokenPair,
		)
		return nil, err
	}
	// validate vpool exists
	if err = k.requireVpool(ctx, pair); err != nil {
		return nil, err
	}

	// validate margin denom
	if msg.Margin.Denom != pair.GetQuoteTokenDenom() {
		err = fmt.Errorf("invalid margin denom")
		k.Logger(ctx).Debug(
			err.Error(),
			"margin_denom",
			msg.Margin.Denom,
			"quote_token_denom",
			pair.GetQuoteTokenDenom(),
		)
		return nil, err
	}

	// ------------- AddMargin -------------
	position, err := k.GetPosition(ctx, pair, msgSender)
	if err != nil {
		k.Logger(ctx).Debug(
			err.Error(),
			"pair",
			pair.String(),
			"trader",
			msg.Sender,
		)
		return nil, err
	}

	remainingMargin, err := k.CalcRemainMarginWithFundingPayment(
		ctx, *position, msg.Margin.Amount.ToDec())
	if err != nil {
		return nil, err
	}

	if !remainingMargin.BadDebt.IsZero() {
		err = fmt.Errorf("failed to add margin; position has bad debt; consider adding more margin")
		k.Logger(ctx).Debug(
			err.Error(),
			"remaining_bad_debt",
			remainingMargin.BadDebt.String(),
		)
		return nil, err
	}

	coinToSend := sdk.NewCoin(pair.GetQuoteTokenDenom(), msg.Margin.Amount)
	if err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx, msgSender, types.VaultModuleAccount, sdk.NewCoins(coinToSend),
	); err != nil {
		k.Logger(ctx).Debug(
			err.Error(),
			"trader",
			msg.Sender,
			"coin",
			coinToSend.String(),
		)
		return nil, err
	}

	position.Margin = remainingMargin.Margin
	position.LastUpdateCumulativePremiumFraction = remainingMargin.LatestCumulativePremiumFraction
	position.BlockNumber = ctx.BlockHeight()
	k.SetPosition(ctx, pair, msgSender, position)

	err = ctx.EventManager().EmitTypedEvent(
		&types.MarginChangedEvent{
			Pair:           pair.String(),
			TraderAddress:  msgSender,
			MarginAmount:   msg.Margin.Amount,
			FundingPayment: remainingMargin.FundingPayment,
		},
	)

	return &types.MsgAddMarginResponse{
		FundingPayment: remainingMargin.FundingPayment,
		Position:       position,
	}, err
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
	traderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// validate margin amount
	if !msg.Margin.Amount.IsPositive() {
		err = fmt.Errorf("margin must be positive, not: %v", msg.Margin.Amount.String())
		k.Logger(ctx).Debug(
			err.Error(),
			"margin_amount",
			msg.Margin.Amount.String(),
		)
		return nil, err
	}

	// validate token pair
	pair, err := common.NewAssetPairFromStr(msg.TokenPair)
	if err != nil {
		k.Logger(ctx).Debug(
			err.Error(),
			"token_pair",
			msg.TokenPair,
		)
		return nil, err
	}

	// validate vpool exists
	if err = k.requireVpool(ctx, pair); err != nil {
		return nil, err
	}

	// validate margin denom
	if msg.Margin.Denom != pair.GetQuoteTokenDenom() {
		err = fmt.Errorf("invalid margin denom")
		k.Logger(ctx).Debug(
			err.Error(),
			"margin_denom",
			msg.Margin.Denom,
			"quote_token_denom",
			pair.GetQuoteTokenDenom(),
		)
		return nil, err
	}

	// ------------- RemoveMargin -------------
	position, err := k.Positions().Get(ctx, pair, traderAddr)
	if err != nil {
		k.Logger(ctx).Debug(
			err.Error(),
			"pair",
			pair.String(),
			"trader",
			msg.Sender,
		)
		return nil, err
	}

	marginDelta := msg.Margin.Amount.Neg()
	remainingMargin, err := k.CalcRemainMarginWithFundingPayment(
		ctx, *position, marginDelta.ToDec())
	if err != nil {
		return nil, err
	}
	if !remainingMargin.BadDebt.IsZero() {
		err = types.ErrFailedRemoveMarginCanCauseBadDebt
		k.Logger(ctx).Debug(
			err.Error(),
			"remaining_bad_debt",
			remainingMargin.BadDebt.String(),
		)
		return nil, err
	}

	position.Margin = remainingMargin.Margin
	position.LastUpdateCumulativePremiumFraction = remainingMargin.LatestCumulativePremiumFraction
	freeCollateral, err := k.calcFreeCollateral(
		ctx, *position, remainingMargin.FundingPayment)
	if err != nil {
		return res, err
	} else if !freeCollateral.IsPositive() {
		return res, fmt.Errorf("not enough free collateral")
	}

	k.Positions().Set(ctx, pair, traderAddr, position)

	coinToSend := sdk.NewCoin(pair.GetQuoteTokenDenom(), msg.Margin.Amount)
	err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.VaultModuleAccount, traderAddr, sdk.NewCoins(coinToSend))
	if err != nil {
		k.Logger(ctx).Debug(
			err.Error(),
			"to",
			msg.Sender,
			"coin",
			coinToSend.String(),
		)
		return nil, err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.MarginChangedEvent{
		Pair:           pair.String(),
		TraderAddress:  traderAddr,
		MarginAmount:   msg.Margin.Amount,
		FundingPayment: remainingMargin.FundingPayment,
	})

	return &types.MsgRemoveMarginResponse{
		MarginOut:      coinToSend,
		FundingPayment: remainingMargin.FundingPayment,
	}, err
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

func (k Keeper) requireVpool(ctx sdk.Context, pair common.AssetPair) (err error) {
	if !k.VpoolKeeper.ExistsPool(ctx, pair) {
		err = fmt.Errorf("%v: %v", types.ErrPairNotFound.Error(), pair.String())
		k.Logger(ctx).Error(
			err.Error(),
			"pair",
			pair.String(),
		)
		return err
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

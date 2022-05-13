package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func (k Keeper) AddMargin(
	ctx sdk.Context,
	pair common.TokenPair,
	trader sdk.AccAddress,
	addedMargin sdk.Dec,
) (err error) {
	position, err := k.Positions().Get(ctx, pair, trader.String())
	if err != nil {
		return err
	}

	position.Margin = position.Margin.Add(addedMargin)

	if err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx,
		trader,
		types.ModuleName,
		sdk.NewCoins(
			sdk.NewCoin(common.StableDenom, addedMargin.TruncateInt()),
		),
	); err != nil {
		return err
	}

	k.Positions().Set(ctx, pair, trader.String(), position)

	return nil
}

func (k Keeper) RemoveMargin(
	goCtx context.Context, msg *types.MsgRemoveMargin,
) (res *types.MsgRemoveMarginResponse, err error) {
	// ------------- Message Setup -------------

	ctx := sdk.UnwrapSDKContext(goCtx)

	// validate trader
	trader, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return res, err
	}

	// validate margin
	margin := msg.Margin.Amount
	if msg.Margin.Denom != common.StableDenom {
		return res, fmt.Errorf("invalid margin denom")
	} else if margin.LTE(sdk.ZeroInt()) {
		return res, fmt.Errorf("margin must be positive, not: %v", margin.String())
	}

	// validate pair
	pair, err := common.NewTokenPairFromStr(msg.Vpool)
	if err != nil {
		return res, err
	}
	err = k.requireVpool(ctx, pair)
	if err != nil {
		return res, err
	}

	// ------------- RemoveMargin -------------

	position, err := k.Positions().Get(ctx, pair, trader.String())
	if err != nil {
		return res, err
	}

	marginDelta := margin.Neg().ToDec()
	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx, *position, marginDelta)
	if err != nil {
		return res, err
	}
	position.Margin = remaining.margin
	position.LastUpdateCumulativePremiumFraction = remaining.latestCPF
	if !remaining.badDebt.IsZero() {
		return res, fmt.Errorf("failed to remove margin; position has bad debt")
	}

	freeCollateral, err := k.calcFreeCollateral(
		ctx, *position, remaining.fPayment, remaining.badDebt)
	if err != nil {
		return res, err
	} else if !freeCollateral.GTE(sdk.ZeroInt()) {
		return res, fmt.Errorf("not enough free collateral")
	}

	k.Positions().Set(ctx, pair, trader.String(), position)

	coinToSend := sdk.NewCoin(common.StableDenom, margin)
	err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.VaultModuleAccount, trader, sdk.NewCoins(coinToSend))
	if err != nil {
		return res, err
	}
	vaultAddr := k.AccountKeeper.GetModuleAddress(types.VaultModuleAccount)

	events.EmitTransfer(ctx,
		/* coin */ coinToSend,
		/* from */ vaultAddr.String(),
		/* to */ trader.String(),
	)

	events.EmitMarginChange(ctx, trader, pair.String(), margin, remaining.fPayment)
	return &types.MsgRemoveMarginResponse{
		MarginOut:      coinToSend,
		FundingPayment: remaining.fPayment,
	}, nil
}

// TODO test: GetMarginRatio
func (k Keeper) GetMarginRatio(
	ctx sdk.Context, position types.Position,
) (sdk.Dec, error) {
	if position.Size_.IsZero() {
		panic("position with zero size") // tODO(mercilex): panic or error? this is a require
	}

	unrealizedPnL, positionNotional, err := k.getPreferencePositionNotionalAndUnrealizedPnL(
		ctx,
		position,
		types.PnLPreferenceOption_MAX,
	)
	if err != nil {
		return sdk.Dec{}, err
	}

	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx,
		/* oldPosition */ position,
		/* marginDelta */ unrealizedPnL,
	)
	if err != nil {
		return sdk.Dec{}, err
	}

	marginRatio := remaining.margin.Sub(remaining.badDebt).Quo(positionNotional)
	return marginRatio, err
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
	switch largerThanOrEqualTo {
	case true:
		if !marginRatio.GTE(baseMarginRatio) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	default:
		if !marginRatio.LT(baseMarginRatio) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	}

	return nil
}

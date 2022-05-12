package keeper

import (
	"fmt"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	ctx sdk.Context,
	pair common.TokenPair,
	trader sdk.AccAddress,
	margin sdk.Int,
) error {
	// require valid token amount
	if margin.IsNegative() {
		return fmt.Errorf("negative margin value: %v", margin.String())
	} else if margin.IsZero() {
		return fmt.Errorf("zero margin in request")
	}

	// require vpool
	err := k.requireVpool(ctx, pair)
	if err != nil {
		return err
	}

	position, err := k.Positions().Get(ctx, pair, trader.String())
	if err != nil {
		return err
	}

	marginDelta := margin.Neg().ToDec()
	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx, pair, position, marginDelta)
	if err != nil {
		return err
	}
	position.Margin = remaining.margin
	position.LastUpdateCumulativePremiumFraction = remaining.latestCPF

	freeCollateral, err := k.calcFreeCollateral(
		ctx, position, remaining.fPayment, remaining.badDebt)
	if err != nil {
		return err
	} else if !freeCollateral.GTE(sdk.ZeroInt()) {
		return fmt.Errorf("not enough free collateral")
	}

	k.Positions().Set(ctx, pair, trader.String(), position)

	coinToSend := sdk.NewCoin(common.StableDenom, margin)
	err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.VaultModuleAccount, trader, sdk.NewCoins(coinToSend))
	if err != nil {
		return err
	}
	vaultAddr := k.AccountKeeper.GetModuleAddress(types.VaultModuleAccount)

	events.EmitTransfer(ctx,
		/* coin */ coinToSend,
		/* from */ vaultAddr.String(),
		/* to */ trader.String(),
	)

	events.EmitMarginChange(ctx, trader, pair.String(), margin, remaining.fPayment)
	return nil
}

// TODO test: GetMarginRatio
func (k Keeper) GetMarginRatio(
	ctx sdk.Context, pair common.TokenPair, trader string,
) (sdk.Dec, error) {
	position, err := k.Positions().Get(ctx, pair, trader) // TODO(mercilex): inefficient position get
	if err != nil {
		return sdk.Dec{}, err
	}

	if position.Size_.IsZero() {
		panic("position with zero size") // tODO(mercilex): panic or error? this is a require
	}

	unrealizedPnL, positionNotional, err := k.getPreferencePositionNotionalAndUnrealizedPnL(
		ctx,
		pair,
		trader,
		types.PnLPreferenceOption_MAX,
	)
	if err != nil {
		return sdk.Dec{}, err
	}

	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx,
		/* pair */ pair,
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

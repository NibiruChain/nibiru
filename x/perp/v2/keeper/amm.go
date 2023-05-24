package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// EditPriceMultiplier edits the peg multiplier of an amm pool after making
// sure there's enough money in the perp EF fund to pay for the repeg. These
// funds get send to the vault to pay for trader's new net margin.
func (k Keeper) EditPriceMultiplier(
	ctx sdk.Context,
	pair asset.Pair,
	newPriceMultiplier sdk.Dec,
) (err error) {
	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		return err
	}

	if newPriceMultiplier.Equal(amm.PriceMultiplier) {
		// same price multiplier, no-op
		return nil
	}

	// Compute cost of re-pegging the pool
	cost, err := amm.CalcRepegCost(newPriceMultiplier)
	if err != nil {
		return err
	}

	err = k.handleMarketUpdateCost(ctx, pair, cost)
	if err != nil {
		return err
	}

	// Do the re-peg
	amm.PriceMultiplier = newPriceMultiplier
	k.AMMs.Insert(ctx, pair, amm)

	_ = ctx.EventManager().EmitTypedEvent(
		&types.AmmUpdatedEvent{
			FinalAmm: amm,
		},
	)

	return
}

// EditSwapInvariant edits the swap invariant of an amm pool after making
// sure there's enough money in the perp EF fund to pay for the repeg. These
// funds get send to the vault to pay for trader's new net margin.
func (k Keeper) EditSwapInvariant(ctx sdk.Context, pair asset.Pair, multiplier sdk.Dec) (err error) {
	// Get the pool
	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		return err
	}

	// Compute cost of re-pegging the pool
	cost, err := amm.CalcUpdateSwapInvariantCost(multiplier)
	if err != nil {
		return err
	}

	err = k.handleMarketUpdateCost(ctx, pair, cost)
	if err != nil {
		return err
	}

	err = amm.UpdateSwapInvariant(multiplier)
	if err != nil {
		return err
	}

	k.AMMs.Insert(ctx, pair, amm)

	_ = ctx.EventManager().EmitTypedEvent(
		&types.AmmUpdatedEvent{
			FinalAmm: amm,
		},
	)

	return
}

func (k Keeper) handleMarketUpdateCost(ctx sdk.Context, pair asset.Pair, costAmt sdk.Int) (err error) {
	if costAmt.IsPositive() {
		// Positive cost, send from perp EF to vault
		cost := sdk.NewCoins(
			sdk.NewCoin(pair.QuoteDenom(), costAmt),
		)
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.PerpEFModuleAccount,
			types.VaultModuleAccount,
			cost,
		)
		if err != nil {
			return types.ErrNotEnoughFundToPayAction.Wrapf(
				"not enough fund in perp ef to pay for repeg, need %s got %s",
				cost.String(),
				k.BankKeeper.GetBalance(ctx, k.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount), pair.QuoteDenom()).String(),
			)
		}
	} else if costAmt.IsNegative() {
		// Negative cost, send from margin vault to perp ef.
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.VaultModuleAccount,
			types.PerpEFModuleAccount,
			sdk.NewCoins(
				sdk.NewCoin(pair.QuoteDenom(), costAmt.Neg()),
			),
		)
		if err != nil { // nolint:staticcheck
			// if there's no money in margin to pay for the repeg, we still repeg. It's surprising if it's
			// happening on mainnet, but it's not a problem.
			// It means there's bad debt in the system, and it's preventing to pay for the repeg down. But the bad debt
			// end up being paid up by the perp EF anyway.
		}
	}
	return nil
}

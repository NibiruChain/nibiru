package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/perp/types/v1"
)

// EditPoolPegMultiplier edits the peg multiplier of an amm pool after making
// sure there's enough money in the perp EF fund to pay for the repeg. These
// funds get send to the vault to pay for trader's new net margin.
func (k Keeper) EditPoolPegMultiplier(
	ctx sdk.Context,
	sender sdk.AccAddress,
	pair asset.Pair,
	pegMultiplier sdk.Dec,
) (err error) {
	// Get the pool
	pool, err := k.PerpAmmKeeper.GetPool(ctx, pair)
	if nil != err {
		return
	}

	// Compute cost of re-pegging the pool
	cost, err := pool.GetRepegCost(pegMultiplier)
	if err != nil {
		return
	}
	costInt := cost.Ceil().TruncateInt()

	err = k.handleMarketUpdateCost(ctx, pair, costInt)
	if err != nil {
		return
	}

	// Do the re-peg
	err = k.PerpAmmKeeper.EditPoolPegMultiplier(ctx, pair, pegMultiplier)
	if err != nil {
		return
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.PegMultiplierUpdate{ // nolint:errcheck
		Pair:   pair,
		NewPeg: pegMultiplier,
		Cost:   costInt,
	})

	return
}

func (k Keeper) EditPoolSwapInvariant(ctx sdk.Context, sender sdk.AccAddress, pair asset.Pair, multiplier sdk.Dec) (err error) {
	// Get the pool
	pool, err := k.PerpAmmKeeper.GetPool(ctx, pair)
	if nil != err {
		return
	}

	// Compute cost of re-pegging the pool
	cost, err := pool.GetSwapInvariantUpdateCost(multiplier)
	if err != nil {
		return
	}
	costInt := cost.Ceil().TruncateInt()

	err = k.handleMarketUpdateCost(ctx, pair, costInt)
	if err != nil {
		return
	}

	// Do the swap invariant update
	pool, err = k.PerpAmmKeeper.EditSwapInvariant(ctx, pair, multiplier)
	if err != nil {
		return
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.SwapInvariantUpdate{ // nolint:errcheck
		Pair:                    pair,
		NewSwapInvariant:        pool.SqrtDepth.Mul(pool.SqrtDepth),
		SwapInvariantMultiplier: multiplier,
		Cost:                    costInt,
	})

	return
}

func (k Keeper) handleMarketUpdateCost(ctx sdk.Context, pair asset.Pair, cost sdk.Int) (err error) {
	if cost.IsPositive() {
		// Positive cost, send from perp EF to vault
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.PerpEFModuleAccount,
			types.VaultModuleAccount,
			sdk.NewCoins(
				sdk.NewCoin(pair.QuoteDenom(), cost),
			),
		)
		if err != nil {
			return types.ErrNotEnoughFundToPayAction
		}
	} else if cost.IsNegative() {
		// Negative cost, send from margin vault to perp ef.
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.VaultModuleAccount,
			types.PerpEFModuleAccount,
			sdk.NewCoins(
				sdk.NewCoin(pair.QuoteDenom(), cost.Neg()),
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

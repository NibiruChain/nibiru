package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// EditPoolPegMultiplier edits the peg multiplier of an amm pool after making sure there's enough money in the perp
// EF fund to pay for the repeg. These funds get send to the vault to pay for trader's new net margin.
func (k Keeper) EditPoolPegMultiplier(
	ctx sdk.Context,
	sender sdk.AccAddress,
	pair asset.Pair,
	pegMultiplier sdk.Dec,
) (err error) {
	if !k.isWhitelisted(ctx, sender) {
		return fmt.Errorf("address is not whitelisted to update peg multiplier: %s", sender)
	}

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

	if costInt.IsPositive() {
		// Positive cost, send from perp EF to vault
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.PerpEFModuleAccount,
			types.VaultModuleAccount,
			sdk.NewCoins(
				sdk.NewCoin(pair.QuoteDenom(), costInt),
			),
		)
		if err != nil {
			return types.ErrNotEnoughFundToPayRepeg
		}
	} else if costInt.IsNegative() {
		// Negative cost, send from margin vault to perp ef.
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.VaultModuleAccount,
			types.PerpEFModuleAccount,
			sdk.NewCoins(
				sdk.NewCoin(pair.QuoteDenom(), costInt.Neg()),
			),
		)
		if err != nil { // nolint:staticcheck
			// if there's no money in margin to pay for the repeg, we still repeg. It's surprising if it's
			// happening on mainnet, but it's not a problem.
			// It means there's bad debt in the system, and it's preventing to pay for the repeg down. But the bad debt
			// end up being paid up by the perp EF anyway.
		}
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

func (k Keeper) isWhitelisted(ctx sdk.Context, addr sdk.AccAddress) bool {
	// TODO(realu): connect that to the admin role in smart contract
	return true
}

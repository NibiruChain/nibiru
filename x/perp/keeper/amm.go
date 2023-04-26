package keeper

import (
	"fmt"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EditPoolPegMultiplier edits the peg multiplier of an amm pool after making sure there's enough money in the perp
// EF fund to pay for the repeg. These funds get send to the vault to pay for trader's new net margin.
func (k Keeper) EditPoolPegMultiplier(ctx sdk.Context, pair asset.Pair, pegMultiplier sdk.Dec) (err error) {
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

	// Do the re-peg
	err = pool.UpdatePeg(pegMultiplier)
	if err != nil {
		return
	}

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
			return fmt.Errorf("not enough fund to pay for repeg")
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
		if err != nil {
			// if there's no money in margin to pay for the repeg, we still repeg. It's surprising if it's
			// happening on mainnet, but it's not a problem.
			// It means there's bad debt in the system, and it's preventing to pay for the repeg down. But the bad debt
			// end up being paid up by the perp EF anyway.
		}
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

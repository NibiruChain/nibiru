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

	// Get the perp EF funds
	ef := k.BankKeeper.GetModuleAccountBalance(ctx, types.PerpEFModuleAccount, pair.QuoteDenom())

	// Check if there's enough money in the perp EF fund to pay for the re-peg
	if ef.Amount.LT(costInt) {
		return fmt.Errorf("not enough fund to pay for repeg")
	}

	// Do the re-peg
	err = pool.UpdatePeg(pegMultiplier)
	if err != nil {
		return
	}

	err = k.BankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.PerpEFModuleAccount,
		types.VaultModuleAccount,
		sdk.NewCoins(
			sdk.NewCoin(pair.QuoteDenom(), costInt),
		),
	)
	if err != nil {
		return
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.PegMultiplierUpdate{ // nolint:errcheck
		Pair:   pair,
		NewPeg: pegMultiplier,
		Cost:   sdk.NewCoin(pair.QuoteDenom(), costInt),
	})

	return
}

func (k Keeper) isWhitelisted(ctx sdk.Context, addr sdk.AccAddress) bool {
	// TODO(realu): connect that to the admin role in smart contract
	return true
}

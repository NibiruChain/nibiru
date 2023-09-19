package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/NibiruChain/collections"
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
	amm, err := k.GetAMM(ctx, pair)
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
	k.SaveAMM(ctx, amm)

	return nil
}

// EditSwapInvariant edits the swap invariant of an amm pool after making
// sure there's enough money in the perp EF fund to pay for the repeg. These
// funds get send to the vault to pay for trader's new net margin.
func (k Keeper) EditSwapInvariant(ctx sdk.Context, pair asset.Pair, newSwapInvariant sdk.Dec) (err error) {
	// Get the pool
	amm, err := k.GetAMM(ctx, pair)
	if err != nil {
		return err
	}

	// Compute cost of re-pegging the pool
	cost, err := amm.CalcUpdateSwapInvariantCost(newSwapInvariant)
	if err != nil {
		return err
	}

	err = k.handleMarketUpdateCost(ctx, pair, cost)
	if err != nil {
		return err
	}

	err = amm.UpdateSwapInvariant(newSwapInvariant)
	if err != nil {
		return err
	}

	k.SaveAMM(ctx, amm)

	return nil
}

func (k Keeper) handleMarketUpdateCost(ctx sdk.Context, pair asset.Pair, costAmt sdkmath.Int) (err error) {
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

// GetMarket returns the market with last version.
func (k Keeper) GetMarket(ctx sdk.Context, pair asset.Pair) (types.Market, error) {
	lastVersion, err := k.MarketLastVersion.Get(ctx, pair)
	if err != nil {
		return types.Market{}, fmt.Errorf("market %s not found", pair)
	}

	market, err := k.Markets.Get(ctx, collections.Join(pair, lastVersion.Version))
	if err != nil {
		return types.Market{}, fmt.Errorf("market %s not found", pair)
	}

	return market, nil
}

// SaveMarket saves the market by pair and version.
func (k Keeper) SaveMarket(ctx sdk.Context, market types.Market) {
	k.Markets.Insert(ctx, collections.Join(market.Pair, market.Version), market)
}

// GetAMM returns the amm with last version.
func (k Keeper) GetAMM(ctx sdk.Context, pair asset.Pair) (types.AMM, error) {
	lastVersion, err := k.MarketLastVersion.Get(ctx, pair)
	if err != nil {
		return types.AMM{}, fmt.Errorf("market %s not found", pair)
	}

	amm, err := k.AMMs.Get(ctx, collections.Join(pair, lastVersion.Version))
	if err != nil {
		return types.AMM{}, fmt.Errorf("market %s not found", pair)
	}

	return amm, nil
}

// SaveAMM saves the amm by pair and version.
func (k Keeper) SaveAMM(ctx sdk.Context, amm types.AMM) {
	k.AMMs.Insert(ctx, collections.Join(amm.Pair, amm.Version), amm)
}

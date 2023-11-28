package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// UnsafeShiftPegMultiplier: [Without checking x/sudo permissions] Edits the peg
// multiplier of an amm pool after making sure there's enough money in the perp
// EF fund to pay for the repeg. These funds get send to the vault to pay for
// trader's new net margin.
func (k Keeper) UnsafeShiftPegMultiplier(
	ctx sdk.Context,
	pair asset.Pair,
	newPriceMultiplier sdk.Dec,
) (err error) {
	amm, err := k.GetAMM(ctx, pair)
	if err != nil {
		return err
	}
	oldPriceMult := amm.PriceMultiplier

	if newPriceMultiplier.Equal(oldPriceMult) {
		// same price multiplier, no-op
		return nil
	}

	// Compute cost of re-pegging the pool
	cost, err := amm.CalcRepegCost(newPriceMultiplier)
	if err != nil {
		return err
	}

	costPaid, err := k.handleMarketUpdateCost(ctx, pair, cost)
	if err != nil {
		return err
	}

	// Do the re-peg
	amm.PriceMultiplier = newPriceMultiplier
	k.SaveAMM(ctx, amm)

	return ctx.EventManager().EmitTypedEvent(&types.EventShiftPegMultiplier{
		OldPegMultiplier: oldPriceMult,
		NewPegMultiplier: newPriceMultiplier,
		CostPaid:         costPaid,
	})
}

// UnsafeShiftSwapInvariant: [Without checking x/sudo permissions] Edit the swap
// invariant of an amm pool after making sure there's enough money in the perp
// fund to pay for the operation. These funds get send to the vault to pay for
// trader's new net margin.
func (k Keeper) UnsafeShiftSwapInvariant(
	ctx sdk.Context, pair asset.Pair, newSwapInvariant sdkmath.Int,
) (err error) {
	// Get the pool
	amm, err := k.GetAMM(ctx, pair)
	if err != nil {
		return err
	}

	cost, err := amm.CalcUpdateSwapInvariantCost(newSwapInvariant.ToLegacyDec())
	if err != nil {
		return err
	}

	costPaid, err := k.handleMarketUpdateCost(ctx, pair, cost)
	if err != nil {
		return err
	}

	err = amm.UpdateSwapInvariant(newSwapInvariant.ToLegacyDec())
	if err != nil {
		return err
	}

	k.SaveAMM(ctx, amm)

	return ctx.EventManager().EmitTypedEvent(&types.EventShiftSwapInvariant{
		OldSwapInvariant: amm.BaseReserve.Mul(amm.QuoteReserve).RoundInt(),
		NewSwapInvariant: newSwapInvariant,
		CostPaid:         costPaid,
	})
}

func (k Keeper) handleMarketUpdateCost(
	ctx sdk.Context, pair asset.Pair, costAmt sdkmath.Int,
) (costPaid sdk.Coin, err error) {
	collateral, err := k.Collateral.Get(ctx)
	if err != nil {
		return costPaid, err
	}

	if costAmt.IsPositive() {
		// Positive cost, send from perp EF to vault
		cost := sdk.NewCoins(
			sdk.NewCoin(collateral, costAmt),
		)
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.PerpEFModuleAccount,
			types.VaultModuleAccount,
			cost,
		)
		if err != nil {
			return costPaid, types.ErrNotEnoughFundToPayAction.Wrapf(
				"need %s, got %s",
				cost.String(),
				k.BankKeeper.GetBalance(ctx, k.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount), collateral).String(),
			)
		} else {
			costPaid = cost[0]
		}
	} else if costAmt.IsNegative() {
		// Negative cost, send from margin vault to perp ef.
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.VaultModuleAccount,
			types.PerpEFModuleAccount,
			sdk.NewCoins(
				sdk.NewCoin(collateral, costAmt.Neg()),
			),
		)
		if err != nil { // nolint:staticcheck
			costPaid = sdk.NewInt64Coin(collateral, 0)
			// Explanation: If there's no money in the vault to pay for the
			// operation, the execution should still be successful. It's
			// surprising if it's happening on mainnet, but it's not a problem.
			// It means there's bad debt in the system, and it's preventing to
			// pay for the repeg down. But the bad debt end up being paid up by
			// the perp EF anyway.
		} else {
			costPaid = sdk.NewCoin(collateral, costAmt.Abs())
		}
	}
	return costPaid, nil
}

// GetMarket returns the market that is enabled. It is the last version of the market.
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

// GetMarketByPairAndVersion this function returns the market by pair and version. It can be enabled or disabled.
func (k Keeper) GetMarketByPairAndVersion(ctx sdk.Context, pair asset.Pair, version uint64) (types.Market, error) {
	market, err := k.Markets.Get(ctx, collections.Join(pair, version))
	if err != nil {
		return types.Market{}, fmt.Errorf("market with pair %s and version %d not found", pair, version)
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

// GetAMMByPairAndVersion returns the amm by pair and version.
func (k Keeper) GetAMMByPairAndVersion(ctx sdk.Context, pair asset.Pair, version uint64) (types.AMM, error) {
	amm, err := k.AMMs.Get(ctx, collections.Join(pair, version))
	if err != nil {
		return types.AMM{}, fmt.Errorf("amm with pair %s and version %d not found", pair, version)
	}

	return amm, nil
}

// SaveAMM saves the amm by pair and version.
func (k Keeper) SaveAMM(ctx sdk.Context, amm types.AMM) {
	k.AMMs.Insert(ctx, collections.Join(amm.Pair, amm.Version), amm)
}

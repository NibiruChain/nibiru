package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// WithdrawFromVault coins from the vault to the receiver.
// If the total amount of coins to withdraw is greater than the vault's amount, then
// withdraw the shortage from the PerpEF and mark it as prepaid bad debt.
//
// Prepaid bad debt will count towards realized bad debt from negative PnL positions
// when those are closed/liquidated.
//
// An example of this happening is when a long position has really high PnL and
// closes their position, realizing their profits.
// There is a counter party short position with really negative PnL, but
// their position hasn't been closed/liquidated yet.
// We must pay the long trader first, which results in funds being taken from the EF.
// when the short position is closed, it also realizes some bad debt but because
// we have already withdrawn from the EF, we don't need to withdraw more from the EF.
//
// args:
// - ctx: context
// - market: the perp market
// - receiver: the receiver of the coins
// - amountToWithdraw: amount of coins to withdraw
//
// returns:
// - error: error
func (k Keeper) WithdrawFromVault(
	ctx sdk.Context,
	market types.Market,
	receiver sdk.AccAddress,
	amountToWithdraw sdkmath.Int,
) (err error) {
	if !amountToWithdraw.IsPositive() {
		return nil
	}

	vaultQuoteBalance := k.BankKeeper.GetBalance(
		ctx,
		k.AccountKeeper.GetModuleAddress(types.VaultModuleAccount),
		market.Pair.QuoteDenom(),
	)
	if vaultQuoteBalance.Amount.LT(amountToWithdraw) {
		// if withdraw amount is larger than entire balance of vault
		// means this trader's profit comes from other under collateral position's future loss
		// and the balance of entire vault is not enough
		// need money from PerpEF to pay first, and record this prepaidBadDebt
		shortage := amountToWithdraw.Sub(vaultQuoteBalance.Amount)
		k.IncrementPrepaidBadDebt(ctx, market, shortage)

		// TODO(k-yang): emit event for prepaid bad debt

		if err := k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.PerpEFModuleAccount,
			types.VaultModuleAccount,
			sdk.NewCoins(
				sdk.NewCoin(market.Pair.QuoteDenom(), shortage),
			),
		); err != nil {
			return err
		}
	}

	// Transfer from Vault to receiver
	return k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		/* from */ types.VaultModuleAccount,
		/* to */ receiver,
		sdk.NewCoins(
			sdk.NewCoin(market.Pair.QuoteDenom(), amountToWithdraw),
		),
	)
}

// IncrementPrepaidBadDebt increases the bad debt for the provided denom.
func (k Keeper) IncrementPrepaidBadDebt(ctx sdk.Context, market types.Market, amount sdkmath.Int) {
	market.PrepaidBadDebt.Amount = market.PrepaidBadDebt.Amount.Add(amount)
	k.SaveMarket(ctx, market)
}

// ZeroPrepaidBadDebt out the prepaid bad debt
func (k Keeper) ZeroPrepaidBadDebt(ctx sdk.Context, market types.Market) {
	market.PrepaidBadDebt.Amount = sdk.ZeroInt()
	k.SaveMarket(ctx, market)
}

// DecrementPrepaidBadDebt decrements the amount of bad debt prepaid by denom.
func (k Keeper) DecrementPrepaidBadDebt(ctx sdk.Context, market types.Market, amount sdkmath.Int) {
	market.PrepaidBadDebt.Amount = market.PrepaidBadDebt.Amount.Sub(amount)
	k.SaveMarket(ctx, market)
}

/*
Realizes the bad debt by first decrementing it from the prepaid bad debt.
Prepaid bad debt accrues when more coins are withdrawn from the vault than the
vault contains, so we "credit" ourselves with prepaid bad debt.

then, when bad debt is actually realized (by closing underwater positions), we
can consume the credit we have built before withdrawing more from the ecosystem fund.
*/
func (k Keeper) realizeBadDebt(ctx sdk.Context, market types.Market, badDebtToRealize sdkmath.Int) (
	err error,
) {
	if market.PrepaidBadDebt.Amount.GTE(badDebtToRealize) {
		// prepaidBadDebtBalance > badDebtToRealize
		k.DecrementPrepaidBadDebt(ctx, market, badDebtToRealize)
	} else {
		// badDebtToRealize > prepaidBadDebtBalance
		k.ZeroPrepaidBadDebt(ctx, market)

		return k.BankKeeper.SendCoinsFromModuleToModule(ctx,
			/*from=*/ types.PerpEFModuleAccount,
			/*to=*/ types.VaultModuleAccount,
			sdk.NewCoins(
				sdk.NewCoin(
					market.Pair.QuoteDenom(),
					badDebtToRealize.Sub(market.PrepaidBadDebt.Amount),
				),
			),
		)
	}

	return nil
}

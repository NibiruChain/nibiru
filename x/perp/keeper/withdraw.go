package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/perp/types"
)

/*
Withdraws coins from the vault to the receiver.
If the total amount of coins to withdraw is greater than the vault's amount, then
withdraw the shortage from the PerpEF and mark it as prepaid bad debt.

Prepaid bad debt will count towards realized bad debt from negative PnL positions
when those are closed/liquidated.

An example of this happening is when a long position has really high PnL and
closes their position, realizing their profits.
There is a counter party short position with really negative PnL, but
their position hasn't been closed/liquidated yet.
We must pay the long trader first, which results in funds being taken from the EF.
When the short position is closed, it also realizes some bad debt but because
we have already withdrawn from the EF, we don't need to withdraw more from the EF.
*/
func (k Keeper) Withdraw(
	ctx sdk.Context,
	denom string,
	receiver sdk.AccAddress,
	amountToWithdraw sdk.Int,
) (err error) {
	if !amountToWithdraw.IsPositive() {
		return nil
	}

	vaultQuoteBalance := k.BankKeeper.GetBalance(
		ctx,
		k.AccountKeeper.GetModuleAddress(types.VaultModuleAccount),
		denom,
	)
	if vaultQuoteBalance.Amount.LT(amountToWithdraw) {
		// if withdraw amount is larger than entire balance of vault
		// means this trader's profit comes from other under collateral position's future loss
		// and the balance of entire vault is not enough
		// need money from PerpEF to pay first, and record this prepaidBadDebt
		shortage := amountToWithdraw.Sub(vaultQuoteBalance.Amount)
		k.PrepaidBadDebtState().Increment(ctx, denom, shortage)
		if err := k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.PerpEFModuleAccount,
			types.VaultModuleAccount,
			sdk.NewCoins(
				sdk.NewCoin(denom, shortage),
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
			sdk.NewCoin(denom, amountToWithdraw),
		),
	)
}

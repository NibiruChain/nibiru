package keeper

import (
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/* transferToInsuranceFund transfer the amount needed to the insurance fund*/
func (k Keeper) transferToInsuranceFund(ctx sdk.Context, trader string, denom string, amount sdk.Int) {
	panic("not implemented")
}

/* withdraw transfer tokens from the trader account to the liquidator account

If withdraw amount is larger than entire balance of vault :
	- means this trader's profit comes from other under collateral position's future loss and the balance of entire vault is not enough
	- need money from IInsuranceFund to pay first, and record this prepaidBadDebt
	- in this case, insurance fund loss must be zero.
*/
func (k Keeper) withdraw(ctx sdk.Context, trader string, liquidator sdk.Address, denom string, amount sdk.Int) {
	moduleAccAddr := k.AccountKeeper.GetModuleAddress(types.ModuleName)
	tokenBalance := k.BankKeeper.GetBalance(ctx, moduleAccAddr, denom)

	if tokenBalance.Amount.LT(amount) {
		missingTokens := tokenBalance
		missingTokens.Amount = amount.Sub(tokenBalance.Amount)

		err := k.withdrawToIF(ctx, missingTokens)
		if err != nil {
			panic(err)
		}
	}

	panic("not implemented")
}

func (k *Keeper) withdrawToIF(ctx sdk.Context, missingTokens sdk.Coin) error {
	panic("not implemented")
}

func (k Keeper) realizeBadDebt(ctx sdk.Context, denom string, badDebt sdk.Int) {
	panic("not implemented")
}

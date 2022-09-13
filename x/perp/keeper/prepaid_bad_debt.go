package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func (k Keeper) IncrementBadDebt(ctx sdk.Context, denom string, amount sdk.Int) {
	pbd := k.PrepaidBadDebt.GetOr(ctx, keys.String(denom), types.PrepaidBadDebt{Amount: sdk.ZeroInt(), Denom: denom})
	k.PrepaidBadDebt.Insert(ctx, keys.String(denom), types.PrepaidBadDebt{
		Denom:  denom,
		Amount: pbd.Amount.Add(amount),
	})
}

func (k Keeper) DecrementBadDebt(ctx sdk.Context, denom string, amount sdk.Int) {
	pbd := k.PrepaidBadDebt.GetOr(ctx, keys.String(denom), types.PrepaidBadDebt{Amount: sdk.ZeroInt(), Denom: denom})
	newPbd := sdk.MaxInt(pbd.Amount.Sub(amount), sdk.ZeroInt())
	k.PrepaidBadDebt.Insert(ctx, keys.String(denom), types.PrepaidBadDebt{
		Denom:  denom,
		Amount: newPbd,
	})
}

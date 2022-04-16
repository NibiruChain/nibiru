package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/MatrixDao/matrix/x/common"
)

func (k Keeper) GetStableSupply(
	ctx sdk.Context,
) sdk.Coin {
	return k.BankKeeper.GetSupply(ctx, common.StableDenom)
}

func (k Keeper) GetGovSupply(
	ctx sdk.Context,
) sdk.Coin {
	return k.BankKeeper.GetSupply(ctx, common.GovDenom)
}

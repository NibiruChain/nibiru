package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

func (k Keeper) GetSupplyNUSD(
	ctx sdk.Context,
) sdk.Coin {
	return k.BankKeeper.GetSupply(ctx, common.StableDenom)
}

func (k Keeper) GetSupplyNIBI(
	ctx sdk.Context,
) sdk.Coin {
	return k.BankKeeper.GetSupply(ctx, common.GovDenom)
}

func (k Keeper) GetStableMarketCap(ctx sdk.Context) sdk.Int {
	return k.GetStableSupply(ctx).Amount
}

func (k Keeper) GetGovMarketCap(ctx sdk.Context) sdk.Int {
	k.DexKeeper.FetchPool()
	return sdk.Int{}
}

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
	return k.GetSupplyNUSD(ctx).Amount
}

func (k Keeper) GetGovMarketCap(ctx sdk.Context) (sdk.Int, error) {
	pairID, err := k.DexKeeper.GetFromPair(ctx, common.GovDenom, common.StableDenom)
	if err != nil {
		return sdk.Int{}, err
	}

	pool := k.DexKeeper.FetchPool(ctx, pairID)

	return sdk.Int{}
}

package keeper

import (
	"github.com/NibiruChain/nibiru/x/common/asset"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) CloseMarket(ctx sdk.Context, pair asset.Pair) error {
	market, err := k.GetMarket(ctx, pair)
	if err != nil {
		return err
	}

	market.Enabled = false
	k.SaveMarket(ctx, market)

	return nil
}

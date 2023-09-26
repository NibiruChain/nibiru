package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

// CloseMarket closes the market. From now on, no new position can be opened on this market or closed.
// Only the open positions can be settled by calling SettlePosition.
func (k Keeper) CloseMarket(ctx sdk.Context, pair asset.Pair) error {
	market, err := k.GetMarket(ctx, pair)
	if err != nil {
		return err
	}

	market.Enabled = false
	k.SaveMarket(ctx, market)

	return nil
}

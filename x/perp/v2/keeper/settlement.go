package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

// CloseMarket closes the market. From now on, no new position can be opened on this market or closed.
// Only the open positions can be settled by calling SettlePosition.
func (k Keeper) CloseMarket(ctx sdk.Context, pair asset.Pair) (err error) {
	market, err := k.GetMarket(ctx, pair)
	if err != nil {
		return err
	}
	amm, err := k.GetAMM(ctx, pair)
	if err != nil {
		return err
	}

	settlementPrice, _, err := amm.ComputeSettlementPrice()
	if err != nil {
		return
	}

	amm.SettlementPrice = settlementPrice
	market.Enabled = false

	k.SaveAMM(ctx, amm)
	k.SaveMarket(ctx, market)

	return nil
}

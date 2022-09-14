package vpool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/x/vpool/keeper"
)

// Called every block to store a snapshot of the vpool.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	for _, pool := range k.GetAllPools(ctx) {
		k.SaveSnapshot(ctx, pool.Pair, pool.QuoteAssetReserve, pool.BaseAssetReserve)
	}
	return []abci.ValidatorUpdate{}
}

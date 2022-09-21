package vpool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/x/vpool/types"

	"github.com/NibiruChain/nibiru/x/vpool/keeper"
)

// EndBlocker Called every block to store a snapshot of the vpool.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	for _, pool := range k.GetAllPools(ctx) {
		snapshot := types.NewReserveSnapshot(
			pool.Pair,
			pool.BaseAssetReserve,
			pool.QuoteAssetReserve,
			ctx.BlockTime(),
			ctx.BlockHeight(),
		)
		k.SaveSnapshot(ctx, snapshot)
	}
	return []abci.ValidatorUpdate{}
}

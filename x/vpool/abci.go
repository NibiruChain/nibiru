package vpool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/x/vpool/keeper"
)

// Called every block to automatically unlock matured locks.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	for _, pool := range k.GetAllPools(ctx) {
		assetPair := pool.Pair.String()
		if err := k.UpdateTWAP(ctx, assetPair); err != nil {
			ctx.Logger().Error("failed to update TWAP", "assetPair", assetPair, "error", err)
		}
	}
	return []abci.ValidatorUpdate{}
}

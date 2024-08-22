package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/set"
)

// IsWhitelistedPair returns existence of a pair in the voting target list
func (k Keeper) IsWhitelistedPair(ctx sdk.Context, pair asset.Pair) bool {
	return k.WhitelistedPairs.Has(ctx, pair)
}

// GetWhitelistedPairs returns the whitelisted pairs list on current vote period
func (k Keeper) GetWhitelistedPairs(ctx sdk.Context) []asset.Pair {
	return k.WhitelistedPairs.Iterate(ctx, collections.Range[asset.Pair]{}).Keys()
}

// refreshWhitelist updates the whitelist by detecting possible changes between
// the current vote targets and the current updated whitelist.
func (k Keeper) refreshWhitelist(ctx sdk.Context, nextWhitelist []asset.Pair, currentWhitelist set.Set[asset.Pair]) {
	updateRequired := false

	if len(currentWhitelist) != len(nextWhitelist) {
		updateRequired = true
	} else {
		for _, pair := range nextWhitelist {
			_, exists := currentWhitelist[pair]
			if !exists {
				updateRequired = true
				break
			}
		}
	}

	if updateRequired {
		for _, p := range k.WhitelistedPairs.Iterate(ctx, collections.Range[asset.Pair]{}).Keys() {
			k.WhitelistedPairs.Delete(ctx, p)
		}
		for _, pair := range nextWhitelist {
			k.WhitelistedPairs.Insert(ctx, pair)
		}
	}
}

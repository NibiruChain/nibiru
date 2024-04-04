package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/set"
)

// IsWhitelistedPair returns existence of a pair in the voting target list
func (k Keeper) IsWhitelistedPair(ctx sdk.Context, pair asset.Pair) bool {
	pairs, _ := k.WhitelistedPairs.Has(ctx, pair)
	return pairs
}

// GetWhitelistedPairs returns the whitelisted pairs list on current vote period
func (k Keeper) GetWhitelistedPairs(ctx sdk.Context) []asset.Pair {
	iter, err := k.WhitelistedPairs.Iterate(ctx, &collections.Range[asset.Pair]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate whitelister pairs", "error", err)
		return nil
	}
	keys, err := iter.Keys()
	if err != nil {
		k.Logger(ctx).Error("failed to get whitelisted pairs keys", "error", err)
		return nil
	}
	return keys
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
		iter, err := k.WhitelistedPairs.Iterate(ctx, &collections.Range[asset.Pair]{})
		if err != nil {
			k.Logger(ctx).Error("failed to iterate whitelister pairs", "error", err)
			return
		}
		keys, err := iter.Keys()
		if err != nil {
			k.Logger(ctx).Error("failed to get whitelisted pairs keys", "error", err)
			return
		}
		for _, p := range keys {
			k.WhitelistedPairs.Remove(ctx, p)
		}
		for _, pair := range nextWhitelist {
			k.WhitelistedPairs.Set(ctx, pair)
		}
	}
}

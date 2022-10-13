package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var p types.Params
	k.paramstore.GetParamSet(ctx, &p)
	return p
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
	k.ActivePairsStore().SetMany(
		ctx, params.Pairs, true)
}

// GetPairs returns the pairs from params
func (k Keeper) GetPairs(ctx sdk.Context) common.AssetPairs {
	var pairs common.AssetPairs
	for _, pair := range k.GetParams(ctx).Pairs {
		pairs = append(pairs, pair)
	}
	return pairs
}

func (k Keeper) GetTwapLookbackWindow(ctx sdk.Context) time.Duration {
	return k.GetParams(ctx).TwapLookbackWindow
}

// GetOraclesForPair returns the oracles for a valid asset pair
func (k Keeper) GetOraclesForPair(ctx sdk.Context, pairID string,
) (oracles []sdk.AccAddress) {
	pair := common.MustNewAssetPair(pairID)
	return k.OraclesStore().Get(ctx, pair)
}

// IsWhitelistedOracle returns true if the address is whitelisted, false if not.
func (k Keeper) IsWhitelistedOracle(
	ctx sdk.Context, pairID string, address sdk.AccAddress,
) bool {
	pair := common.MustNewAssetPair(pairID)
	oracles := k.OraclesStore().Get(ctx, pair)
	for _, addr := range oracles {
		if addr.Equals(address) {
			return true
		}
	}
	return false
}

// GetPair returns the market if it is in the pricefeed system
func (k Keeper) IsActivePair(ctx sdk.Context, pairID string) bool {
	pair := common.MustNewAssetPair(pairID)
	return k.ActivePairsStore().Get(ctx, pair)
}

// GetOraclesForPairs returns the 'oraclesMatrix' corresponding to 'pairs'.
// 'oraclesMap' is a map from pair → list of oracles.
// This function effectively gives a subset of the OraclesState KVStore.
func (k Keeper) GetOraclesForPairs(ctx sdk.Context, pairs common.AssetPairs,
) map[common.AssetPair][]sdk.AccAddress {
	oraclesMap := make(map[common.AssetPair][]sdk.AccAddress)
	for _, pair := range pairs {
		oraclesMap[pair] = k.GetOraclesForPair(ctx, pair.String())
	}
	return oraclesMap
}

// Whitelists given 'oracles' for all of the current pairs in the module params.
func (k Keeper) WhitelistOracles(ctx sdk.Context, oracles []sdk.AccAddress) {
	startParams := k.GetParams(ctx)
	for _, pair := range startParams.Pairs {
		k.addOraclesForPair(ctx, pair, oracles)
	}
}

// addOraclesForPair returns a 'newPair', which has an Oracles field formed by
// the unique set union of 'oracles' and 'pair.Oracles'.
func (k Keeper) addOraclesForPair(
	ctx sdk.Context, pair common.AssetPair, oracles []sdk.AccAddress,
) {
	startingOracles := k.OraclesStore().Get(ctx, pair)
	var endingOracles []sdk.AccAddress
	uniquePairOracles := make(map[string]bool)
	for _, oracle := range append(startingOracles, oracles...) {
		if _, found := uniquePairOracles[(oracle.String())]; !found {
			endingOracles = append(endingOracles, oracle)
			uniquePairOracles[oracle.String()] = true
		}
	}
	k.OraclesStore().Set(ctx, pair, endingOracles)
}

// WhitelistOraclesForPairs whitelists 'oracles' for the given 'pairs'.
func (k Keeper) WhitelistOraclesForPairs(
	ctx sdk.Context, oracles []sdk.AccAddress, proposedPairs []common.AssetPair,
) {
	paramsPairs := k.GetPairs(ctx)

	newPairs := []common.AssetPair{}
	for _, pair := range proposedPairs {
		pairIDBytes := []byte(pair.String())
		if !k.OraclesStore().getKV(ctx).Has(pairIDBytes) {
			newPairs = append(newPairs, pair)
		}

		k.OraclesStore().AddOracles(ctx, pair, oracles)
	}
	endingPairs := append(paramsPairs, newPairs...)
	k.SetParams(ctx, types.NewParams(endingPairs, k.GetTwapLookbackWindow(ctx)))
}

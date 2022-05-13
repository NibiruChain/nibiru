package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

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
}

// GetPairs returns the markets from params
func (k Keeper) GetPairs(ctx sdk.Context) types.Pairs {
	return k.GetParams(ctx).Pairs
}

// GetOracles returns the oracles in the pricefeed store
func (k Keeper) GetOracles(ctx sdk.Context, pairID string) ([]sdk.AccAddress, error) {
	for _, m := range k.GetPairs(ctx) {
		if pairID == m.PairID() {
			return m.Oracles, nil
		}
	}
	return nil, sdkerrors.Wrap(types.ErrInvalidPair, pairID)
}

// GetOracle returns the oracle from the store or an error if not found
func (k Keeper) GetOracle(
	ctx sdk.Context, pairID string, address sdk.AccAddress,
) (sdk.AccAddress, error) {
	oracles, err := k.GetOracles(ctx, pairID)
	if err != nil {
		// Error already wrapped
		return nil, err
	}
	for _, addr := range oracles {
		if addr.Equals(address) {
			return addr, nil
		}
	}
	return nil, sdkerrors.Wrap(types.ErrInvalidOracle, address.String())
}

// GetPair returns the market if it is in the pricefeed system
func (k Keeper) GetPair(ctx sdk.Context, pairID string) (types.Pair, bool) {
	markets := k.GetPairs(ctx)

	for i := range markets {
		if markets[i].PairID() == pairID {
			return markets[i], true
		}
	}
	return types.Pair{}, false
}

// GetAuthorizedAddresses returns a list of addresses that have special authorization within this module, eg the oracles of all markets.
func (k Keeper) GetAuthorizedAddresses(ctx sdk.Context) []sdk.AccAddress {
	var oracles []sdk.AccAddress
	uniqueOracles := map[string]bool{}

	for _, m := range k.GetPairs(ctx) {
		for _, o := range m.Oracles {
			// de-dup list of oracles
			if _, found := uniqueOracles[o.String()]; !found {
				oracles = append(oracles, o)
			}
			uniqueOracles[o.String()] = true
		}
	}
	return oracles
}

// Whitelists given 'oracles' for all of the current pairs.
func (k Keeper) WhitelistOracles(ctx sdk.Context, oracles []sdk.AccAddress) {
	startParams := k.GetParams(ctx)
	var endPairs types.Pairs
	for _, pair := range startParams.Pairs {
		var pairOracles []sdk.AccAddress
		uniquePairOracles := make(map[string]bool)
		for _, o := range append(pair.Oracles, oracles...) {
			if _, found := uniquePairOracles[(o.String())]; !found {
				pairOracles = append(pairOracles, o)
				uniquePairOracles[o.String()] = true
			}
		}
		endPairs = append(endPairs,
			types.Pair{Token0: pair.Token0, Token1: pair.Token1,
				Oracles: pairOracles, Active: pair.Active})
	}
	endParams := types.NewParams(endPairs)
	k.SetParams(ctx, endParams)
}

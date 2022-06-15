package keeper

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

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
		endPair := appendOraclesToPair(pair, oracles)
		endPairs = append(endPairs, endPair)
	}
	endParams := types.NewParams(endPairs)
	k.SetParams(ctx, endParams)
}

// appendOraclesToPair returns a 'newPair', which has an Oracles field formed by
// the unique set union of 'oracles' and 'pair.Oracles'.
func appendOraclesToPair(pair types.Pair, oracles []sdk.AccAddress) (newPair types.Pair) {
	var pairOracles []sdk.AccAddress
	uniquePairOracles := make(map[string]bool)
	for _, oracle := range append(pair.Oracles, oracles...) {
		if _, found := uniquePairOracles[(oracle.String())]; !found {
			pairOracles = append(pairOracles, oracle)
			uniquePairOracles[oracle.String()] = true
		}
	}

	// sort the oracles to make reads faster w/ binary search
	pairOraclesStrings := []string{}
	for _, oracle := range pairOracles {
		pairOraclesStrings = append(pairOraclesStrings, oracle.String())
	}
	sort.Strings(pairOraclesStrings)
	pairOracles = []sdk.AccAddress{}
	for _, oracleStr := range pairOraclesStrings {
		pairOracles = append(pairOracles, sdk.MustAccAddressFromBech32(oracleStr))
	}

	return types.Pair{
		Token0: pair.Token0, Token1: pair.Token1,
		Oracles: pairOracles, Active: pair.Active}
}

// WhitelistOracleForPairs whitelists 'oracles' for the given 'pairs'.
func (k Keeper) WhitelistOraclesForPairs(
	ctx sdk.Context, oracles []sdk.AccAddress, pairs []string,
) error {
	paramsPairs := k.GetPairs(ctx)

	// Contained in params check
	paramsIdxProposedIdxMap := make(map[int]int) // maps paramsIdx -> proposedIdx
	// proposedIdx: index of the proposed pairs array
	// paramsIdx: and the params pairs array to avoid unnecessary looping
	for proposedIdx, pair := range pairs {
		proposedPair, err := common.NewAssetPairFromStr(pair)
		if err != nil {
			return err
		}

		found, paramsIdx := paramsPairs.ContainsAtIndex(types.Pair{
			Token0: proposedPair.Token0,
			Token1: proposedPair.Token1,
		})
		paramsIdxProposedIdxMap[paramsIdx] = proposedIdx

		if !found {
			return fmt.Errorf("pair %v:%v not contained in params",
				proposedPair.Token0, proposedPair.Token1)
			// Refactor
			// TODO create sdkerror for this
			// NOTE Q: For reviewer, should we allow this to pass instead of throwing an error?
			// That would make this function open up new pairs?
			// Or, should that be a separate governance proposal?
			// It seems convenient to be able to whitelist pairs by intializing them
			// with oracles.
		}
	}

	var endingParamsPairs types.Pairs
	for paramIdx, paramPair := range paramsPairs {

		var endingPair types.Pair
		_, found := paramsIdxProposedIdxMap[paramIdx]

		if !found {
			endingPair = paramPair
		} else {
			endingPair = appendOraclesToPair(paramPair, oracles)
		}

		endingParamsPairs = append(endingParamsPairs, endingPair)
	}

	endParams := types.NewParams(endingParamsPairs)
	k.SetParams(ctx, endParams)
	return nil
}

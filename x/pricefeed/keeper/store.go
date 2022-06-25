package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// OraclesStore maintains the "oracles" KVStore: maps pair name → oracles.
func (k Keeper) OraclesStore() OraclesState {
	return (OraclesState)(k)
}

// ActivePairsStore maintains the "active pairs" KVStore: maps pair name → isActive.
// If a pair doesn't have a key in the store, the pair is inactive.
func (k Keeper) ActivePairsStore() ActivePairsState {
	return (ActivePairsState)(k)
}

//-----------------------------------------------------------------------------
// OraclesState implements methods for updating the "oracles" sdk.KVStore
type OraclesState Keeper

var oraclesNamespace = []byte("oracles")

func (state OraclesState) getKV(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(state.storeKey), oraclesNamespace)
}

func (state OraclesState) Get(
	ctx sdk.Context, pair common.AssetPair,
) (oracles []sdk.AccAddress) {
	kvStore := state.getKV(ctx)
	key := []byte(pair.String())
	valueBytes := kvStore.Get(key)
	if valueBytes == nil {
		return []sdk.AccAddress{}
	}

	oraclesMarshaler := &types.OraclesMarshaler{}
	state.cdc.MustUnmarshal(
		/*bytes=*/ valueBytes,
		/*codec.ProtoMarshaler=*/ oraclesMarshaler)

	return oraclesMarshaler.Oracles
}

func (state OraclesState) Set(
	ctx sdk.Context, pair common.AssetPair, oracles []sdk.AccAddress,
) {
	key := []byte(pair.String())
	kvStore := state.getKV(ctx)
	oraclesMarshaler := &types.OraclesMarshaler{Oracles: oracles}
	kvStore.Set(key, state.cdc.MustMarshal(oraclesMarshaler))
}

func (state OraclesState) AddOracles(
	ctx sdk.Context, pair common.AssetPair, oracles []sdk.AccAddress,
) {
	startOracles := state.Get(ctx, pair)
	endOracles := append(startOracles, oracles...)
	state.Set(ctx, pair, endOracles)
}

func (state OraclesState) Iterate(
	ctx sdk.Context,
	do func(*types.OraclesMarshaler) (stop bool),
) {
	kvStore := state.getKV(ctx)
	iter := kvStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		oraclesMarshaler := &types.OraclesMarshaler{}
		state.cdc.MustUnmarshal(iter.Value(), oraclesMarshaler)
		if !do(oraclesMarshaler) {
			break
		}
	}
}

//-----------------------------------------------------------------------------
// ActivePairsState implements methods for updating the "active pairs" sdk.KVStore
type ActivePairsState Keeper

var activePairsNamespace = []byte("active pairs")

func (state ActivePairsState) getKV(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(state.storeKey), activePairsNamespace)
}

func (state ActivePairsState) Get(
	ctx sdk.Context, pair common.AssetPair,
) (active bool) {
	kvStore := state.getKV(ctx)
	key := []byte(pair.String())
	valueBytes := kvStore.Get(key)
	if valueBytes == nil {
		return false
	}

	activePairsMarshaler := &types.ActivePairMarshaler{}
	state.cdc.MustUnmarshal(
		/*bytes=*/ valueBytes,
		/*codec.ProtoMarshaler=*/ activePairsMarshaler)

	isActive := activePairsMarshaler.IsActive
	if (valueBytes != nil) && !isActive {
		kvStore.Delete(key)
	}
	return isActive
}

/* ActivePairsState.Set either sets a pair to active or deletes it from the
key-value store (i.e., pairs default to inactive if they don't exist). */
func (state ActivePairsState) Set(
	ctx sdk.Context, pair common.AssetPair, active bool,
) {
	key := []byte(pair.String())
	kvStore := state.getKV(ctx)
	if active {
		activePairsMarshaler := &types.ActivePairMarshaler{IsActive: active}
		kvStore.Set(key, state.cdc.MustMarshal(activePairsMarshaler))
	} else if !active && kvStore.Has(key) {
		kvStore.Delete(key)
	} // else {do nothing}
}

func (state ActivePairsState) SetMany(
	ctx sdk.Context, pairs common.AssetPairs, active bool,
) {
	for _, pair := range pairs {
		state.Set(ctx, pair, active)
	}
}

func (state ActivePairsState) AddActivePairs(
	ctx sdk.Context, pairs []common.AssetPair,
) {
	for _, pair := range pairs {
		state.Set(ctx, pair, true)
	}
}

func (state ActivePairsState) Iterate(
	ctx sdk.Context,
	do func(*types.ActivePairMarshaler) (stop bool),
) {
	kvStore := state.getKV(ctx)
	iter := kvStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		activePairsMarshaler := &types.ActivePairMarshaler{}
		state.cdc.MustUnmarshal(iter.Value(), activePairsMarshaler)
		if !do(activePairsMarshaler) {
			break
		}
	}
}

package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// RawPrices initializes and returns RawPriceState
func (k Keeper) RawPrices(ctx sdk.Context) RawPricesState {
	store := ctx.KVStore(k.storeKey)
	return RawPricesState{
		cdc:       k.cdc,
		ctx:       ctx,
		rawPrices: prefix.NewStore(store, types.RawPricesObjectsPrefix),
	}
}

// OraclesState implements methods for updating the "oracles" sdk.KVStore
type OraclesState Keeper

var oraclesNamespace = []byte("oracles")

func (s OraclesState) getKV(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(s.storeKey), oraclesNamespace)
}

func (s OraclesState) pairKV(ctx sdk.Context, pair common.AssetPair) sdk.KVStore {
	prefixKey := append([]byte(pair.String()), 0x00) // null terminated to avoid prefix overlaps of BTC/USD -> BTC/USDT
	return prefix.NewStore(s.getKV(ctx), prefixKey)
}

func (s OraclesState) Get(
	ctx sdk.Context, pair common.AssetPair,
) (oracles []sdk.AccAddress) {
	iter := s.pairKV(ctx, pair).Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		oracles = append(oracles, iter.Key())
	}

	return oracles
}

func (s OraclesState) AddOracles(
	ctx sdk.Context, pair common.AssetPair, oracles []sdk.AccAddress,
) {

	kvStore := s.pairKV(ctx, pair)

	for _, oracle := range oracles {
		kvStore.Set(oracle, []byte{})
	}
}

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

// Set either sets a pair to active or deletes it from the
// key-value store (i.e., pairs default to inactive if they don't exist). */
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

type RawPricesState struct {
	cdc       codec.BinaryCodec
	ctx       sdk.Context
	rawPrices sdk.KVStore
}

func (r RawPricesState) PrimaryKeyRaw(pair, oracle string) []byte {
	// TODO(mercilex): can be made more efficient by precomputing length

	k1 := append([]byte(pair), 0x00)
	k2 := append([]byte(oracle), 0x00)

	return append(k1, k2...)
}

func (r RawPricesState) PairStore(pair common.AssetPair) sdk.KVStore {
	k1 := append([]byte(pair.String()), 0x00)

	return prefix.NewStore(r.rawPrices, k1)
}

func (r RawPricesState) PrimaryKey(p *types.PostedPrice) []byte {
	return r.PrimaryKeyRaw(p.PairID, p.Oracle)
}

func (r RawPricesState) Create(p *types.PostedPrice) {
	key := r.PrimaryKey(p)
	r.rawPrices.Set(key, r.cdc.MustMarshal(p))
}

func (r RawPricesState) GetForPair(pair common.AssetPair) types.PostedPrices {
	var prices types.PostedPrices

	r.IterateForPair(pair, func(p *types.PostedPrice) (stop bool) {
		prices = append(prices, *p)
		return false
	})

	return prices
}

func (r RawPricesState) IterateForPair(pair common.AssetPair, do func(p *types.PostedPrice) (stop bool)) {
	iter := r.PairStore(pair).Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		pp := new(types.PostedPrice)
		r.cdc.MustUnmarshal(iter.Value(), pp)
		if do(pp) {
			return
		}
	}
}

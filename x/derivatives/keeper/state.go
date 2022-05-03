package keeper

import (
	"errors"
	"fmt"

	v1 "github.com/NibiruChain/nibiru/x/derivatives/types/v1"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	errNotFound = errors.New("not found")
)

func (k Keeper) Params() ParamsState {
	return (ParamsState)(k)
}

func (k Keeper) Positions() PositionsState {
	return (PositionsState)(k)
}

func (k Keeper) PairMetadata() PairMetadata {
	return (PairMetadata)(k)
}

func (k Keeper) Whitelist() Whitelist {
	return (Whitelist)(k)
}

var paramsNamespace = []byte{0x0}
var paramsKey = []byte{0x0}

type ParamsState Keeper

func (p ParamsState) getKV(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(p.key), paramsNamespace)
}

func (p ParamsState) Get(ctx sdk.Context) (*v1.Params, error) {
	kv := p.getKV(ctx)

	value := kv.Get(paramsKey)
	if value == nil {
		return nil, fmt.Errorf("not found")
	}

	params := new(v1.Params)
	p.cdc.MustUnmarshal(value, params)
	return params, nil
}

func (p ParamsState) Set(ctx sdk.Context, params *v1.Params) {
	kv := p.getKV(ctx)
	kv.Set(paramsKey, p.cdc.MustMarshal(params))
}

var positionsNamespace = []byte{0x1}

type PositionsState Keeper

func (p PositionsState) getKV(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(p.key), positionsNamespace)
}

func (p PositionsState) keyFromType(position *v1.Position) []byte {
	return p.keyFromRaw(position.Pair, position.Address)
}

func (p PositionsState) keyFromRaw(pair, address string) []byte {
	// TODO(mercilex): not sure if namespace overlap safe | update(mercilex) it is not overlap safe
	return []byte(pair + address)
}

func (p PositionsState) Create(ctx sdk.Context, position *v1.Position) error {
	key := p.keyFromType(position)
	kv := p.getKV(ctx)
	if kv.Has(key) {
		return fmt.Errorf("already exists")
	}

	kv.Set(key, p.cdc.MustMarshal(position))
	return nil
}

func (p PositionsState) Get(ctx sdk.Context, pair, address string) (*v1.Position, error) {
	kv := p.getKV(ctx)

	key := p.keyFromRaw(pair, address)
	valueBytes := kv.Get(key)
	if valueBytes == nil {
		return nil, errNotFound
	}

	position := new(v1.Position)
	p.cdc.MustUnmarshal(valueBytes, position)

	return position, nil
}

func (p PositionsState) Update(ctx sdk.Context, position *v1.Position) error {
	kv := p.getKV(ctx)
	key := p.keyFromType(position)

	if !kv.Has(key) {
		return errNotFound
	}

	kv.Set(key, p.cdc.MustMarshal(position))
	return nil
}

func (p PositionsState) Set(ctx sdk.Context, position *v1.Position) {
	p.getKV(ctx).Set(p.keyFromType(position), p.cdc.MustMarshal(position))
}

var pairMetadataNamespace = []byte{0x2}

type PairMetadata Keeper

func (p PairMetadata) getKV(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(p.key), pairMetadataNamespace)
}

func (p PairMetadata) Get(ctx sdk.Context, pair string) (*v1.PairMetadata, error) {
	kv := p.getKV(ctx)

	v := kv.Get([]byte(pair))
	if v == nil {
		return nil, errNotFound
	}

	pairMetadata := new(v1.PairMetadata)
	p.cdc.MustUnmarshal(v, pairMetadata)

	return pairMetadata, nil
}

func (p PairMetadata) Set(ctx sdk.Context, metadata *v1.PairMetadata) {
	kv := p.getKV(ctx)
	kv.Set([]byte(metadata.Pair), p.cdc.MustMarshal(metadata))
}

var whitelistNamespace = []byte{0x3}

type Whitelist Keeper

func (w Whitelist) getKV(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(w.key), whitelistNamespace)
}

func (w Whitelist) IsWhitelisted(ctx sdk.Context, address string) bool {
	kv := w.getKV(ctx)

	return kv.Has([]byte(address))
}

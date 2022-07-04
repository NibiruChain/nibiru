package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/x/perp/types"
)

func (k Keeper) Params(
	goCtx context.Context, req *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

func (k Keeper) PositionsState(ctx sdk.Context) PositionsState {
	return newPositions(ctx, k.storeKey, k.cdc)
}

func (k Keeper) PairMetadataState(ctx sdk.Context) PairMetadataState {
	return newPairMetadata(ctx, k.storeKey, k.cdc)
}

func (k Keeper) WhitelistState(ctx sdk.Context) WhitelistState {
	return newWhitelist(ctx, k.storeKey, k.cdc)
}

func (k Keeper) PrepaidBadDebtState(ctx sdk.Context) PrepaidBadDebtState {
	return newPrepaidBadDebtState(ctx, k.storeKey, k.cdc)
}

var paramsNamespace = []byte{0x0}
var paramsKey = []byte{0x0}

type ParamsState Keeper

func (p ParamsState) getKV(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(p.storeKey), paramsNamespace)
}

func (p ParamsState) Get(ctx sdk.Context) (*types.Params, error) {
	kv := p.getKV(ctx)

	value := kv.Get(paramsKey)
	if value == nil {
		return nil, fmt.Errorf("not found")
	}

	params := new(types.Params)
	p.cdc.MustUnmarshal(value, params)
	return params, nil
}

func (p ParamsState) Set(ctx sdk.Context, params *types.Params) {
	kv := p.getKV(ctx)
	kv.Set(paramsKey, p.cdc.MustMarshal(params))
}

var positionsNamespace = []byte{0x1}

type PositionsState struct {
	positions sdk.KVStore
	cdc       codec.BinaryCodec
}

func newPositions(ctx sdk.Context, key sdk.StoreKey, cdc codec.BinaryCodec) PositionsState {
	return PositionsState{
		positions: prefix.NewStore(ctx.KVStore(key), positionsNamespace),
		cdc:       cdc,
	}
}

func (p PositionsState) keyFromType(position *types.Position) []byte {
	traderAddress, err := sdk.AccAddressFromBech32(position.TraderAddress)
	if err != nil {
		panic(err)
	}
	return p.keyFromRaw(position.Pair, traderAddress)
}

func (p PositionsState) keyFromRaw(pair common.AssetPair, address sdk.AccAddress) []byte {
	// TODO(mercilex): not sure if namespace overlap safe | update(mercilex) it is not overlap safe
	return []byte(pair.String() + address.String())
}

func (p PositionsState) Create(position *types.Position) error {
	key := p.keyFromType(position)
	if p.positions.Has(key) {
		return fmt.Errorf("already exists")
	}

	p.positions.Set(key, p.cdc.MustMarshal(position))
	return nil
}

func (p PositionsState) Get(pair common.AssetPair, traderAddr sdk.AccAddress) (*types.Position, error) {
	key := p.keyFromRaw(pair, traderAddr)
	valueBytes := p.positions.Get(key)
	if valueBytes == nil {
		return nil, types.ErrPositionNotFound
	}

	position := new(types.Position)
	p.cdc.MustUnmarshal(valueBytes, position)

	return position, nil
}

func (p PositionsState) Update(position *types.Position) error {
	key := p.keyFromType(position)

	if !p.positions.Has(key) {
		return types.ErrPositionNotFound
	}

	p.positions.Set(key, p.cdc.MustMarshal(position))
	return nil
}

func (p PositionsState) Set(pair common.AssetPair, traderAddr sdk.AccAddress, position *types.Position) {
	positionID := p.keyFromRaw(pair, traderAddr)
	p.positions.Set(positionID, p.cdc.MustMarshal(position))
}

func (p PositionsState) Iterate(do func(position *types.Position) (stop bool)) {
	iter := p.positions.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		position := new(types.Position)
		p.cdc.MustUnmarshal(iter.Value(), position)
		if !do(position) {
			break
		}
	}
}

func (p PositionsState) Delete(pair common.AssetPair, addr sdk.AccAddress) error {
	primaryKey := p.keyFromRaw(pair, addr)

	if !p.positions.Has(primaryKey) {
		return types.ErrPositionNotFound.Wrapf("in pair %s", pair)
	}
	p.positions.Delete(primaryKey)

	return nil
}

var pairMetadataNamespace = []byte{0x2}

func newPairMetadata(ctx sdk.Context, key sdk.StoreKey, cdc codec.BinaryCodec) PairMetadataState {
	store := ctx.KVStore(key)
	return PairMetadataState{
		pairsMetadata: prefix.NewStore(store, pairMetadataNamespace),
		cdc:           cdc,
	}
}

type PairMetadataState struct {
	pairsMetadata sdk.KVStore
	cdc           codec.BinaryCodec
}

func (p PairMetadataState) Get(pair common.AssetPair) (*types.PairMetadata, error) {
	v := p.pairsMetadata.Get([]byte(pair.String()))
	if v == nil {
		return nil, types.ErrPairMetadataNotFound
	}

	pairMetadata := new(types.PairMetadata)
	p.cdc.MustUnmarshal(v, pairMetadata)

	return pairMetadata, nil
}

func (p PairMetadataState) Set(metadata *types.PairMetadata) {
	p.pairsMetadata.Set([]byte(metadata.Pair.String()), p.cdc.MustMarshal(metadata))
}

func (p PairMetadataState) GetAll() []*types.PairMetadata {
	iterator := p.pairsMetadata.Iterator(nil, nil)

	var pairMetadatas []*types.PairMetadata
	for ; iterator.Valid(); iterator.Next() {
		var pairMetadata = new(types.PairMetadata)
		p.cdc.MustUnmarshal(iterator.Value(), pairMetadata)
		pairMetadatas = append(pairMetadatas, pairMetadata)
	}

	return pairMetadatas
}

var whitelistNamespace = []byte{0x3}

type WhitelistState struct {
	whitelists sdk.KVStore
	cdc        codec.BinaryCodec
}

func newWhitelist(ctx sdk.Context, key sdk.StoreKey, cdc codec.BinaryCodec) WhitelistState {
	return WhitelistState{
		whitelists: prefix.NewStore(ctx.KVStore(key), whitelistNamespace),
		cdc:        cdc,
	}
}

func (w WhitelistState) IsWhitelisted(address sdk.AccAddress) bool {
	return w.whitelists.Has(address)
}

func (w WhitelistState) Add(address sdk.AccAddress) {
	w.whitelists.Set(address, []byte{})
}

func (w WhitelistState) Iterate(do func(addr sdk.AccAddress) (stop bool)) {
	iter := w.whitelists.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		if !do(iter.Key()) {
			break
		}
	}
}

var prepaidBadDebtNamespace = []byte{0x4}

type PrepaidBadDebtState struct {
	prepaidBadDebt sdk.KVStore
}

func newPrepaidBadDebtState(ctx sdk.Context, key sdk.StoreKey, _ codec.BinaryCodec) PrepaidBadDebtState {
	return PrepaidBadDebtState{
		prepaidBadDebt: prefix.NewStore(ctx.KVStore(key), prepaidBadDebtNamespace),
	}
}

// Get Fetches the amount of bad debt prepaid by denom. Returns zero if the denom is not found.
func (s PrepaidBadDebtState) Get(denom string) (amount sdk.Int) {
	v := s.prepaidBadDebt.Get([]byte(denom))
	if v == nil {
		return sdk.ZeroInt()
	}

	err := amount.Unmarshal(v)
	if err != nil {
		panic(err)
	}

	return amount
}

// Iterate iterates over known prepaid bad debt.
func (s PrepaidBadDebtState) Iterate(do func(denom string, amount sdk.Int) (stop bool)) {
	iter := s.prepaidBadDebt.Iterator(nil, nil)

	for ; iter.Valid(); iter.Next() {
		amount := sdk.Int{}
		err := amount.Unmarshal(iter.Value())
		if err != nil {
			panic(err)
		}
		if !do(string(iter.Key()), amount) {
			break
		}
	}
}

// Set sets the amount of bad debt prepaid by denom.
func (s PrepaidBadDebtState) Set(denom string, amount sdk.Int) {
	b, err := amount.Marshal()
	if err != nil {
		panic(err)
	}
	s.prepaidBadDebt.Set([]byte(denom), b)
}

// Increment increments the amount of bad debt prepaid by denom.
// Calling this method on a denom that doesn't exist is effectively the same as setting the amount (0 + increment).
func (s PrepaidBadDebtState) Increment(denom string, increment sdk.Int) (amount sdk.Int) {
	amount = s.Get(denom).Add(increment)

	b, err := amount.Marshal()
	if err != nil {
		panic(err)
	}
	s.prepaidBadDebt.Set([]byte(denom), b)

	return amount
}

// Decrement decrements the amount of bad debt prepaid by denom.
// The lowest it can be decremented to is zero. Trying to decrement a prepaid bad
// debt balance to below zero will clip it at zero.
func (s PrepaidBadDebtState) Decrement(denom string, decrement sdk.Int) (amount sdk.Int) {
	amount = sdk.MaxInt(s.Get(denom).Sub(decrement), sdk.ZeroInt())

	b, err := amount.Marshal()
	if err != nil {
		panic(err)
	}
	s.prepaidBadDebt.Set([]byte(denom), b)

	return amount
}

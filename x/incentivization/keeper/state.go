package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/incentivization/types"
)

const (
	IncentivizationProgramStartID uint64 = 0
)

var (
	incentivizationProgramNamespace      = []byte{0x0}
	incentivizationProgramIDNamespace    = append(incentivizationProgramNamespace, 0x0)
	incentivizationProgramIDKey          = []byte{0x1}
	incentiviationProgramObjectNamespace = append(incentivizationProgramNamespace, 0x1)
	incentivizationProgramDenomIndex     = append(incentivizationProgramNamespace, 0x2)
	incentivizationProgramDenomMap       = append(incentivizationProgramNamespace, 0x3)
)

func (k Keeper) IncentivizationProgramsState(ctx sdk.Context) IncentivizationProgramState {
	return newIncentivizationProgramState(ctx, k.storeKey, k.cdc)
}

func newIncentivizationProgramState(ctx sdk.Context, key sdk.StoreKey, cdc codec.Codec) IncentivizationProgramState {
	store := ctx.KVStore(key)
	return IncentivizationProgramState{
		cdc:                                cdc,
		ctx:                                ctx,
		programID:                          prefix.NewStore(store, incentivizationProgramIDNamespace),
		incentivizationPrograms:            prefix.NewStore(store, incentiviationProgramObjectNamespace),
		denomToIncentivizationProgramIndex: prefix.NewStore(store, incentivizationProgramDenomIndex),
		denomMap:                           prefix.NewStore(store, incentivizationProgramDenomMap),
	}
}

type IncentivizationProgramState struct {
	cdc codec.Codec
	ctx sdk.Context

	programID                          sdk.KVStore
	incentivizationPrograms            sdk.KVStore // maps objects
	denomToIncentivizationProgramIndex sdk.KVStore // maps denom to incentivization program
	denomMap                           sdk.KVStore // provides the current list of incentivized denomss through a map
}

// PeekNextID returns the next ID without actually increasing the counter.
func (s IncentivizationProgramState) PeekNextID() uint64 {
	id := s.programID.Get(incentivizationProgramIDKey)
	switch id {
	case nil:
		return 0
	default:
		return sdk.BigEndianToUint64(id)
	}
}

func (s IncentivizationProgramState) Create(program *types.IncentivizationProgram) {
	if program.Id != 0 {
		panic("incentivization program id must not be set")
	}
	id := s.nextPrimaryKey()
	pk := sdk.Uint64ToBigEndian(id) // TODO(mercilex): inefficient, doing this twice

	program.Id = id
	s.incentivizationPrograms.Set(pk, s.cdc.MustMarshal(program))

	s.index(pk, program)
}

func (s IncentivizationProgramState) Get(id uint64) (*types.IncentivizationProgram, error) {
	bytes := s.incentivizationPrograms.Get(sdk.Uint64ToBigEndian(id))
	if bytes == nil {
		return nil, types.ErrIncentivizationProgramNotFound.Wrapf("%d", id)
	}

	program := new(types.IncentivizationProgram)
	s.cdc.MustUnmarshal(bytes, program)
	return program, nil
}

func (s IncentivizationProgramState) index(pk []byte, program *types.IncentivizationProgram) {
	s.denomMap.Set([]byte(program.LpDenom), []byte{})
	s.denomToIncentivizationProgramIndex.Set(s.denomKey(program.LpDenom, pk), []byte{})
}

/*func (s IncentivizationProgramState) unindex(pk []byte, program *types.IncentivizationProgram) {
	s.denomToIncentivizationProgramIndex.Delete(s.denomKey(program.LpDenom, pk))
	// now we check if there are more lp denoms
	iter := s.denomToIncentivizationProgramIndex.Iterator(s.denomKey(program.LpDenom, nil), nil)
	defer iter.Close()
	// in case the iter is not valid, it means that there are no more
	// incentivization programs associated with the given denom.
	// Hence we clear the denom map.
	if !iter.Valid() {
		s.denomMap.Delete([]byte(program.LpDenom))
	}
}*/

func (s IncentivizationProgramState) nextPrimaryKey() uint64 {
	idBytes := s.programID.Get(incentivizationProgramIDKey)
	var id uint64

	switch idBytes {
	case nil:
		id = IncentivizationProgramStartID
	default:
		id = sdk.BigEndianToUint64(idBytes)
	}

	s.programID.Set(incentivizationProgramIDKey, sdk.Uint64ToBigEndian(id+1))

	return id
}

func (s IncentivizationProgramState) denomKey(denom string, pk []byte) []byte {
	key := make([]byte, 0, len(denom)+1+len(pk))
	key = append(key, []byte(denom)...)
	key = append(key, 0xFF)
	key = append(key, pk...)
	return key
}

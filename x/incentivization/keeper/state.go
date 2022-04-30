package keeper

import (
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	cdc                                codec.Codec
	ctx                                sdk.Context
	programID                          sdk.KVStore
	incentivizationPrograms            sdk.KVStore // maps objects
	denomToIncentivizationProgramIndex sdk.KVStore // maps denom to incentivization program
	denomMap                           sdk.KVStore // provides the current list of incentivized denomss through a map
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

func (s IncentivizationProgramState) index(pk []byte, program *types.IncentivizationProgram) {
	s.denomMap.Set([]byte(program.LpDenom), []byte{})
	s.denomToIncentivizationProgramIndex.Set(s.denomKey(program.LpDenom, pk), []byte{})
}

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

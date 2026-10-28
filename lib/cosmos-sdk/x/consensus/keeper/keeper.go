// Package keeper is the keeper of the Cosmos SDK x/consensus module.
//
// Deprecated: NibiruApp no longer uses this module. Nibiru's own x/consensus
// module (github.com/NibiruChain/nibiru/v2/x/consensus) owns the "consensus"
// store, serves the Params query, and rejects MsgUpdateParams, because the
// x-consensus contract owns the consensus params. The types moved to
// github.com/NibiruChain/nibiru/v2/x/consensus/types, which this package
// imports. It is kept only for lib's own depinject test apps (the
// testutil/app_config.go files and testutil/configurator) and the ibc-go
// simapps, and will be removed together with that test wiring.
package keeper

import (
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	storetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/consensus/exported"
	"github.com/NibiruChain/nibiru/v2/x/consensus/types"
)

var _ exported.ConsensusParamSetter = (*Keeper)(nil)

type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	authority string
}

func NewKeeper(cdc codec.BinaryCodec, storeKey storetypes.StoreKey, authority string) Keeper {
	return Keeper{
		storeKey:  storeKey,
		cdc:       cdc,
		authority: authority,
	}
}

func (k *Keeper) GetAuthority() string {
	return k.authority
}

// Get gets the consensus parameters
func (k *Keeper) Get(ctx sdk.Context) (*tmproto.ConsensusParams, error) {
	store := ctx.KVStore(k.storeKey)

	cp := &tmproto.ConsensusParams{}
	bz := store.Get(types.ParamStoreKeyConsensusParams)

	if err := k.cdc.Unmarshal(bz, cp); err != nil {
		return nil, err
	}

	return cp, nil
}

func (k *Keeper) Has(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)

	return store.Has(types.ParamStoreKeyConsensusParams)
}

// Set sets the consensus parameters
func (k *Keeper) Set(ctx sdk.Context, cp *tmproto.ConsensusParams) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ParamStoreKeyConsensusParams, k.cdc.MustMarshal(cp))
}

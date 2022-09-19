package keeper

import (
	"fmt"

	"github.com/NibiruChain/nibiru/collections"
	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/NibiruChain/nibiru/x/common"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/perp/types"
)

// PositionsIndexes provides the indexes used py perp positions.
type PositionsIndexes struct {
	// Address tracks positions to the address owning the position.
	Address *collections.MultiIndex[keys.StringKey, keys.Pair[common.AssetPair, keys.StringKey], types.Position]
}

// IndexList implements the collections.Indexes interface.
func (p PositionsIndexes) IndexList() []collections.Index[keys.Pair[common.AssetPair, keys.StringKey], types.Position] {
	return []collections.Index[keys.Pair[common.AssetPair, keys.StringKey], types.Position]{p.Address}
}

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      sdk.StoreKey
	ParamSubspace paramtypes.Subspace

	BankKeeper      types.BankKeeper
	AccountKeeper   types.AccountKeeper
	PricefeedKeeper types.PricefeedKeeper
	VpoolKeeper     types.VpoolKeeper
	EpochKeeper     types.EpochKeeper

	Positions collections.IndexedMap[keys.Pair[common.AssetPair, keys.StringKey], types.Position, *types.Position, PositionsIndexes]
}

// NewKeeper Creates a new x/perp Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	paramSubspace paramtypes.Subspace,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	priceKeeper types.PricefeedKeeper,
	vpoolKeeper types.VpoolKeeper,
	epochKeeper types.EpochKeeper,
) Keeper {
	// Ensure that the module account is set.
	if moduleAcc := accountKeeper.GetModuleAddress(types.ModuleName); moduleAcc == nil {
		panic("The x/perp module account has not been set")
	}

	// Set param.types.'KeyTable' if it has not already been set
	if !paramSubspace.HasKeyTable() {
		paramSubspace = paramSubspace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		ParamSubspace: paramSubspace,

		AccountKeeper:   accountKeeper,
		BankKeeper:      bankKeeper,
		PricefeedKeeper: priceKeeper,
		VpoolKeeper:     vpoolKeeper,
		EpochKeeper:     epochKeeper,

		Positions: collections.NewIndexedMap[keys.Pair[common.AssetPair, keys.StringKey], types.Position, *types.Position, PositionsIndexes](cdc, storeKey, 3, PositionsIndexes{
			Address: collections.NewMultiIndex[keys.StringKey, keys.Pair[common.AssetPair, keys.StringKey]](func(v types.Position) keys.StringKey {
				// this function returns an indexing key containing the trader address.
				return keys.String(v.TraderAddress)
			}),
		}),
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetParams get all parameters as types.Params
func (k *Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.ParamSubspace.GetParamSet(ctx, &params)
	return params
}

// SetParams set the params
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.ParamSubspace.SetParamSet(ctx, &params)
}

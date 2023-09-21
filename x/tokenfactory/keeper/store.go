package keeper

import (
	"github.com/NibiruChain/collections"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tftypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

// StoreAPI isolates the collections for the x/tokenfactory module.
// Ultimately, the denoms are registered in the x/bank module if valid. Because
// of this, a denom cannot be deleted once it exists.
//
// The StoreAPI hides private methods to make the developer experience less
// error-prone when working on the module.
type StoreAPI struct {
	// Denoms: IndexedMap
	//  - primary key (PK): Token factory denom (TFDenom) as a string
	//  - value (V): TFDenom payload with validation
	//  - indexers (I): Indexed by creator for easy querying
	Denoms       collections.IndexedMap[storePKType, storeVType, IndexesTokenFactory]
	ModuleParams collections.Item[tftypes.ModuleParams]
	creator      collections.KeySet[storePKType]
	denomAdmins  collections.Map[storePKType, tftypes.DenomAuthorityMetadata]
	bankKeeper   tftypes.BankKeeper
}

func (api StoreAPI) InsertDenom(
	ctx sdk.Context, denom tftypes.TFDenom,
) error {
	if err := denom.Validate(); err != nil {
		return err
	}
	key := denom.String()
	// The x/bank keeper is the source of truth.
	found := api.HasDenom(ctx, denom)
	if found {
		return tftypes.ErrDenomAlreadyRegistered.Wrap(key)
	}

	api.Denoms.Insert(ctx, key, denom)
	api.creator.Insert(ctx, denom.Creator)

	// The denom creator is the default admin. The admin can be updated after
	// the denom exists using the ChangeAdmin message. Having no admin is also
	// allowed.
	api.bankKeeper.SetDenomMetaData(ctx, denom.DefaultBankMetadata())
	api.denomAdmins.Insert(ctx, key, tftypes.DenomAuthorityMetadata{
		Admin: denom.Creator,
	})
	return nil
}

func (api StoreAPI) MustInsertDenom(
	ctx sdk.Context, denom tftypes.TFDenom,
) {
	if err := api.InsertDenom(ctx, denom); err != nil {
		panic(err)
	}
}

// InsertDenomGenesis_NoBankUpdate: Populates the x/tokenfactory state without
// making any assumptions about the x/bank state. This function is unsafe and
// should only be used in InitGenesis or upgrades that populate state from an
// exported genesis.
func (api StoreAPI) InsertDenomGenesis_NoBankUpdate(
	ctx sdk.Context, genDenom tftypes.GenesisDenom,
) {
	denom := tftypes.DenomStr(genDenom.Denom).MustToStruct()
	admin := genDenom.AuthorityMetadata.Admin
	key := denom.String()
	api.Denoms.Insert(ctx, key, denom)
	api.creator.Insert(ctx, denom.Creator)
	api.denomAdmins.Insert(ctx, key, tftypes.DenomAuthorityMetadata{
		Admin: admin,
	})
}

// HasDenom: True if the denom has already been registered.
func (api StoreAPI) HasDenom(
	ctx sdk.Context, denom tftypes.TFDenom,
) bool {
	_, found := api.bankKeeper.GetDenomMetaData(ctx, denom.String())
	return found
}

func (api StoreAPI) HasCreator(ctx sdk.Context, creator string) bool {
	return api.creator.Has(ctx, creator)
}

// GetDenomAuthorityMetadata returns the authority metadata for a specific denom
func (api StoreAPI) GetDenomAuthorityMetadata(
	ctx sdk.Context, denom string,
) (tftypes.DenomAuthorityMetadata, error) {
	return api.denomAdmins.Get(ctx, denom)
}

// ---------------------------------------------
// StoreAPI - Under the hood
// ---------------------------------------------

type storePKType = string
type storeVType = tftypes.TFDenom

// NewTFDenomStore: Creates an indexed map over token facotry denoms indexed
// by creator address.
func NewTFDenomStore(
	storeKey storetypes.StoreKey, cdc sdkcodec.BinaryCodec,
) collections.IndexedMap[storePKType, storeVType, IndexesTokenFactory] {
	primaryKeyEncoder := collections.StringKeyEncoder
	valueEncoder := collections.ProtoValueEncoder[tftypes.TFDenom](cdc)

	var namespace collections.Namespace = tftypes.KeyPrefixDenom
	var namespaceCreatorIdx collections.Namespace = tftypes.KeyPrefixCreatorIndexer

	return collections.NewIndexedMap[storePKType, storeVType](
		storeKey, namespace, primaryKeyEncoder, valueEncoder,
		IndexesTokenFactory{
			Creator: collections.NewMultiIndex[string, storePKType, storeVType](
				storeKey, namespaceCreatorIdx,
				collections.StringKeyEncoder, // index key (IK)
				collections.StringKeyEncoder, // primary key (PK)
				func(v tftypes.TFDenom) string { return v.Creator },
			),
		},
	)
}

// IndexesTokenFactory: Abstraction for indexing over the TF denom store.
type IndexesTokenFactory struct {
	// Creator MultiIndex:
	//  - indexing key (IK): bech32 address of the creator of TF denom.
	//  - primary key (PK): full TF denom of the form 'factory/{creator}/{subdenom}'
	//  - value (V): struct version of TF denom with validate function
	Creator collections.MultiIndex[string, string, storeVType]
}

func (idxs IndexesTokenFactory) IndexerList() []collections.Indexer[string, storeVType] {
	return []collections.Indexer[string, storeVType]{
		idxs.Creator,
	}
}

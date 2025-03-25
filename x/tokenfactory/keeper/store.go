package keeper

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"

	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tftypes "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
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
	// The x/bank keeper is the source of truth.
	key := denom.Denom()
	found := api.HasDenom(ctx, denom)
	if found {
		return tftypes.ErrDenomAlreadyRegistered.Wrap(key.String())
	}

	admin := denom.Creator
	api.unsafeInsertDenom(ctx, denom, admin)

	api.bankKeeper.SetDenomMetaData(ctx, denom.DefaultBankMetadata())
	api.denomAdmins.Set(ctx, key.String(), tftypes.DenomAuthorityMetadata{
		Admin: admin,
	})
	return nil
}

// unsafeInsertDenom: Adds a token factory denom to state with the given admin.
// NOTE: unsafe → assumes pre-validated inputs
func (api StoreAPI) unsafeInsertDenom(
	ctx sdk.Context, denom tftypes.TFDenom, admin string,
) {
	denomStr := denom.Denom()
	api.Denoms.Set(ctx, denomStr.String(), denom)
	api.creator.Set(ctx, denom.Creator)
	api.bankKeeper.SetDenomMetaData(ctx, denom.DefaultBankMetadata())
	api.denomAdmins.Set(ctx, denomStr.String(), tftypes.DenomAuthorityMetadata{
		Admin: admin,
	})
	_ = ctx.EventManager().EmitTypedEvent(&tftypes.EventCreateDenom{
		Denom:   denomStr.String(),
		Creator: denom.Creator,
	})
}

// unsafeGenesisInsertDenom: Populates the x/tokenfactory state without
// making any assumptions about the x/bank state. This function should only be
// used in InitGenesis or upgrades that populate state from an exported genesis.
// NOTE: unsafe → assumes pre-validated inputs
func (api StoreAPI) unsafeGenesisInsertDenom(
	ctx sdk.Context, genDenom tftypes.GenesisDenom,
) {
	denom := tftypes.DenomStr(genDenom.Denom).MustToStruct()
	admin := genDenom.AuthorityMetadata.Admin
	api.unsafeInsertDenom(ctx, denom, admin)
}

// HasDenom: True if the denom has already been registered.
func (api StoreAPI) HasDenom(
	ctx sdk.Context, denom tftypes.TFDenom,
) bool {
	_, found := api.bankKeeper.GetDenomMetaData(ctx, denom.Denom().String())
	return found
}

func (api StoreAPI) HasCreator(ctx sdk.Context, creator string) (bool, error) {
	return api.creator.Has(ctx, creator)
}

// GetDenomAuthorityMetadata returns the admin (authority metadata) for a
// specific denom. This differs from the x/bank metadata.
func (api StoreAPI) GetDenomAuthorityMetadata(
	ctx sdk.Context, denom string,
) (tftypes.DenomAuthorityMetadata, error) {
	metadata, err := api.denomAdmins.Get(ctx, denom)
	if err != nil {
		return metadata, tftypes.ErrGetAdmin.Wrap(err.Error())
	}
	return metadata, nil
}

func (api StoreAPI) GetAdmin(
	ctx sdk.Context, denom string,
) (string, error) {
	metadata, err := api.denomAdmins.Get(ctx, denom)
	if err != nil {
		return "", err
	}
	return metadata.Admin, nil
}

// ---------------------------------------------
// StoreAPI - Under the hood
// ---------------------------------------------

type (
	storePKType = string
	storeVType  = tftypes.TFDenom
)

// NewTFDenomStore: Creates an indexed map over token facotry denoms indexed
// by creator address.
func NewTFDenomStore(
	sb *collections.SchemaBuilder, cdc sdkcodec.BinaryCodec,
) collections.IndexedMap[storePKType, storeVType, IndexesTokenFactory] {
	primaryKeyEncoder := collections.StringKey
	valueEncoder := sdkcodec.CollValue[tftypes.TFDenom](cdc)

	return *collections.NewIndexedMap[storePKType, storeVType, IndexesTokenFactory](
		sb, tftypes.KeyPrefixDenom.Prefix(), "denoms", primaryKeyEncoder, valueEncoder,
		IndexesTokenFactory{
			Creator: *indexes.NewMulti[string, storePKType, storeVType](
				sb, tftypes.KeyPrefixCreatorIndexer.Prefix(), "creator",
				collections.StringKey, // index key (IK)
				collections.StringKey, // primary key (PK)
				func(pk storePKType, value storeVType) (string, error) { return value.Creator, nil },
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
	Creator indexes.Multi[string, string, storeVType]
}

func (idxs IndexesTokenFactory) IndexesList() []collections.Index[string, storeVType] {
	return []collections.Index[string, storeVType]{
		&idxs.Creator,
	}
}

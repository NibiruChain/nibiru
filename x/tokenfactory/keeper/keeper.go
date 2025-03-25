package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/log"

	corestoretypes "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"

	tftypes "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

// Keeper of this module maintains collections of feeshares for contracts
// registered to receive Nibiru Chain gas fees.
type Keeper struct {
	storeService corestoretypes.KVStoreService
	storeKey     storetypes.StoreKey
	cdc          codec.BinaryCodec

	Store StoreAPI

	// interfaces with other modules
	bankKeeper          tftypes.BankKeeper
	accountKeeper       tftypes.AccountKeeper
	communityPoolKeeper tftypes.CommunityPoolKeeper

	// the address capable of executing a MsgUpdateParams message. Typically,
	// this should be the x/gov module account.
	authority string
}

// NewKeeper: creates a Keeper instance for the module.
func NewKeeper(
	storeService corestoretypes.KVStoreService,
	cdc codec.BinaryCodec,
	bk tftypes.BankKeeper,
	ak tftypes.AccountKeeper,
	communityPoolKeeper tftypes.CommunityPoolKeeper,
	authority string,
) Keeper {
	if addr := ak.GetModuleAddress(tftypes.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", tftypes.ModuleName))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		storeService: storeService,
		Store: StoreAPI{
			Denoms: NewTFDenomStore(sb, cdc),
			ModuleParams: collections.NewItem(
				sb, tftypes.KeyPrefixModuleParams.Prefix(), "params",
				codec.CollValue[tftypes.ModuleParams](cdc),
			),
			creator: collections.NewKeySet(
				sb, tftypes.KeyPrefixCreator.Prefix(), "creator",
				collections.StringKey,
			),
			denomAdmins: collections.NewMap[storePKType, tftypes.DenomAuthorityMetadata](
				sb, tftypes.KeyPrefixDenomAdmin.Prefix(), "denom_admins",
				collections.StringKey,
				codec.CollValue[tftypes.DenomAuthorityMetadata](cdc),
			),
			bankKeeper: bk,
		},
		cdc:                 cdc,
		bankKeeper:          bk,
		accountKeeper:       ak,
		communityPoolKeeper: communityPoolKeeper,
		authority:           authority,
	}
	return k
}

// GetAuthority returns the x/feeshare module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", fmt.Sprintf("x/%s", tftypes.ModuleName))
}

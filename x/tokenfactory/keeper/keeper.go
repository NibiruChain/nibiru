package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	tftypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

// Keeper of this module maintains collections of feeshares for contracts
// registered to receive Nibiru Chain gas fees.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

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
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	bk tftypes.BankKeeper,
	ak tftypes.AccountKeeper,
	communityPoolKeeper tftypes.CommunityPoolKeeper,
	authority string,
) Keeper {
	return Keeper{
		storeKey: storeKey,
		Store: StoreAPI{
			Denoms: NewTFDenomStore(storeKey, cdc),
			ModuleParams: collections.NewItem(
				storeKey, tftypes.KeyPrefixModuleParams,
				collections.ProtoValueEncoder[tftypes.ModuleParams](cdc),
			),
			creator: collections.NewKeySet[string](
				storeKey, tftypes.KeyPrefixCreator,
				collections.StringKeyEncoder,
			),
			denomAdmins: collections.NewMap[storePKType, tftypes.DenomAuthorityMetadata](
				storeKey, tftypes.KeyPrefixDenomAdmin,
				collections.StringKeyEncoder,
				collections.ProtoValueEncoder[tftypes.DenomAuthorityMetadata](cdc),
			),
			bankKeeper: bk,
		},
		cdc:                 cdc,
		bankKeeper:          bk,
		accountKeeper:       ak,
		communityPoolKeeper: communityPoolKeeper,
		authority:           authority,
	}
}

// GetAuthority returns the x/feeshare module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", tftypes.ModuleName))
}

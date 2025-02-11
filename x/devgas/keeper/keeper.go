package keeper

import (
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	devgas "github.com/NibiruChain/nibiru/v2/x/devgas"
)

// Keeper of this module maintains collections of feeshares for contracts
// registered to receive Nibiru Chain gas fees.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	bankKeeper    devgas.BankKeeper
	wasmKeeper    wasmkeeper.Keeper
	accountKeeper devgas.AccountKeeper

	// feeCollectorName is the name of x/auth module's fee collector module
	// account, "fee_collector", which collects transaction fees for distribution
	// to all stakers.
	//
	// See the `[AllocateTokens]` function from x/distribution to learn more.
	// [AllocateTokens]: https://github.com/cosmos/cosmos-sdk/blob/v0.50.3/x/distribution/keeper/allocation.go
	feeCollectorName string

	// DevGasStore: IndexedMap
	//  - primary key (PK): Contract address. The contract is the primary key
	//  because there's exactly one deployer and withdrawer.
	//  - value (V): FeeShare value saved into state.
	//  - indexers (I):  Indexed by deployer and withdrawer
	DevGasStore collections.IndexedMap[string, devgas.FeeShare, DevGasIndexes]

	ModuleParams collections.Item[devgas.ModuleParams]

	// the address capable of executing a MsgUpdateParams message. Typically,
	// this should be the x/gov module account.
	authority string
}

// NewKeeper creates new instances of the fees Keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	bk devgas.BankKeeper,
	wk wasmkeeper.Keeper,
	ak devgas.AccountKeeper,
	feeCollector string,
	authority string,
) Keeper {
	return Keeper{
		storeKey:         storeKey,
		cdc:              cdc,
		bankKeeper:       bk,
		wasmKeeper:       wk,
		accountKeeper:    ak,
		feeCollectorName: feeCollector,
		authority:        authority,
		DevGasStore:      NewDevGasStore(storeKey, cdc),
		ModuleParams: collections.NewItem(
			storeKey, devgas.KeyPrefixParams,
			collections.ProtoValueEncoder[devgas.ModuleParams](cdc),
		),
	}
}

// GetAuthority returns the x/feeshare module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", devgas.ModuleName))
}

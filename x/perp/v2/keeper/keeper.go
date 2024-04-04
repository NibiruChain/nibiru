package keeper

import (
	"fmt"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/cosmos/cosmos-sdk/runtime"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	BankKeeper    bankkeeper.Keeper
	AccountKeeper authkeeper.AccountKeeper
	OracleKeeper  types.OracleKeeper
	EpochKeeper   types.EpochKeeper
	SudoKeeper    types.SudoKeeper

	MarketLastVersion collections.Map[asset.Pair, types.MarketLastVersion]
	Markets           collections.Map[collections.Pair[asset.Pair, uint64], types.Market]
	AMMs              collections.Map[collections.Pair[asset.Pair, uint64], types.AMM]
	Collateral        collections.Item[string]

	Positions              collections.Map[collections.Pair[collections.Pair[asset.Pair, uint64], sdk.AccAddress], types.Position]
	ReserveSnapshots       collections.Map[collections.Pair[asset.Pair, time.Time], types.ReserveSnapshot]
	DnREpoch               collections.Item[uint64]                                                    // Keeps track of the current DnR epoch.
	DnREpochName           collections.Item[string]                                                    // Keeps track of the current DnR epoch identifier, provided by x/epoch.
	GlobalVolumes          collections.Map[uint64, math.Int]                                           // Keeps track of global volumes for each epoch.
	TraderVolumes          collections.Map[collections.Pair[sdk.AccAddress, uint64], math.Int]         // Keeps track of user volumes for each epoch.
	GlobalDiscounts        collections.Map[math.Int, math.LegacyDec]                                   // maps a volume level to a discount
	TraderDiscounts        collections.Map[collections.Pair[sdk.AccAddress, math.Int], math.LegacyDec] // maps a user and volume level to a discount, supersedes global discounts
	EpochRebateAllocations collections.Map[uint64, types.DNRAllocation]                                // maps an epoch to a string representing the allocation of rebates for that epoch
}

// NewKeeper Creates a new x/perp Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey *storetypes.KVStoreKey,

	accountKeeper authkeeper.AccountKeeper,
	bankKeeper bankkeeper.Keeper,
	oracleKeeper types.OracleKeeper,
	epochKeeper types.EpochKeeper,
	sudoKeeper types.SudoKeeper,
) Keeper {
	// Ensure that the module account is set.
	if moduleAcc := accountKeeper.GetModuleAddress(types.ModuleName); moduleAcc == nil {
		panic("The x/perp module account has not been set")
	}

	storeService := runtime.NewKVStoreService(storeKey)
	sb := collections.NewSchemaBuilder(storeService)

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		BankKeeper:    bankKeeper,
		AccountKeeper: accountKeeper,
		OracleKeeper:  oracleKeeper,
		EpochKeeper:   epochKeeper,
		SudoKeeper:    sudoKeeper,
		MarketLastVersion: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespaceMarketLastVersion)),
			storeKey.String(),
			asset.PairKeyEncoder,
			codec.CollValue[types.MarketLastVersion](cdc),
		),
		Markets: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespaceMarkets)),
			storeKey.String(),
			collections.PairKeyCodec(asset.PairKeyEncoder, collections.Uint64Key),
			codec.CollValue[types.Market](cdc),
		),
		AMMs: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespaceAmms)),
			storeKey.String(),
			collections.PairKeyCodec(asset.PairKeyEncoder, collections.Uint64Key),
			codec.CollValue[types.AMM](cdc),
		),
		Positions: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespacePositions)),
			storeKey.String(),
			collections.PairKeyCodec(collections.PairKeyCodec(asset.PairKeyEncoder, collections.Uint64Key), sdk.AccAddressKey),
			codec.CollValue[types.Position](cdc),
		),
		ReserveSnapshots: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespaceReserveSnapshots)),
			storeKey.String(),
			collections.PairKeyCodec(asset.PairKeyEncoder, sdk.TimeKey),
			codec.CollValue[types.ReserveSnapshot](cdc),
		),
		DnREpoch: collections.NewItem(
			sb,
			collections.NewPrefix(int(NamespaceDnrEpoch)),
			storeKey.String(),
			collections.Uint64Value,
		),
		GlobalVolumes: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespaceGlobalVolumes)),
			storeKey.String(),
			collections.Uint64Key,
			sdk.IntValue,
		),
		TraderVolumes: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespaceUserVolumes)),
			storeKey.String(),
			collections.PairKeyCodec(sdk.AccAddressKey, collections.Uint64Key),
			sdk.IntValue,
		),
		GlobalDiscounts: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespaceGlobalDiscounts)),
			storeKey.String(),
			common.SdkIntKey,
			common.LegacyDecValue,
		),
		TraderDiscounts: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespaceUserDiscounts)),
			storeKey.String(),
			collections.PairKeyCodec(sdk.AccAddressKey, common.SdkIntKey),
			common.LegacyDecValue,
		),
		EpochRebateAllocations: collections.NewMap(
			sb,
			collections.NewPrefix(int(NamespaceRebatesAllocations)),
			storeKey.String(),
			collections.Uint64Key,
			codec.CollValue[types.DNRAllocation](cdc),
		),
		Collateral: collections.NewItem(
			sb,
			collections.NewPrefix(int(NamespaceCollateral)),
			storeKey.String(),
			collections.StringValue,
		),
		DnREpochName: collections.NewItem(
			sb,
			collections.NewPrefix(int(NamespaceDnrEpochName)),
			storeKey.String(),
			collections.StringValue,
		),
	}
}

const (
	NamespaceMarkets uint8 = iota + 11 // == 11 because iota starts from 0
	NamespaceAmms
	NamespacePositions
	NamespaceReserveSnapshots
	NamespaceDnrEpoch
	NamespaceGlobalVolumes
	NamespaceUserVolumes
	NamespaceGlobalDiscounts
	NamespaceUserDiscounts
	NamespaceRebatesAllocations
	NamespaceMarketLastVersion
	NamespaceCollateral
	NamespaceDnrEpochName
)

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

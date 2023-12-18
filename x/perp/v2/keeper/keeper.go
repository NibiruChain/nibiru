package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/NibiruChain/collections"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	BankKeeper    types.BankKeeper
	AccountKeeper types.AccountKeeper
	OracleKeeper  types.OracleKeeper
	EpochKeeper   types.EpochKeeper
	SudoKeeper    types.SudoKeeper

	// Extends the Keeper with admin functions. See admin.go.
	Admin admin

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
	storeKey storetypes.StoreKey,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	oracleKeeper types.OracleKeeper,
	epochKeeper types.EpochKeeper,
	sudoKeeper types.SudoKeeper,
) Keeper {
	// Ensure that the module account is set.
	if moduleAcc := accountKeeper.GetModuleAddress(types.ModuleName); moduleAcc == nil {
		panic("The x/perp module account has not been set")
	}

	k := Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		BankKeeper:    bankKeeper,
		AccountKeeper: accountKeeper,
		OracleKeeper:  oracleKeeper,
		EpochKeeper:   epochKeeper,
		SudoKeeper:    sudoKeeper,
		MarketLastVersion: collections.NewMap(
			storeKey, NamespaceMarketLastVersion,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[types.MarketLastVersion](cdc),
		),
		Markets: collections.NewMap(
			storeKey, NamespaceMarkets,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.Uint64KeyEncoder),
			collections.ProtoValueEncoder[types.Market](cdc),
		),
		AMMs: collections.NewMap(
			storeKey, NamespaceAmms,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.Uint64KeyEncoder),
			collections.ProtoValueEncoder[types.AMM](cdc),
		),
		Positions: collections.NewMap(
			storeKey, NamespacePositions,
			collections.PairKeyEncoder(collections.PairKeyEncoder(asset.PairKeyEncoder, collections.Uint64KeyEncoder), collections.AccAddressKeyEncoder),
			collections.ProtoValueEncoder[types.Position](cdc),
		),
		ReserveSnapshots: collections.NewMap(
			storeKey, NamespaceReserveSnapshots,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.TimeKeyEncoder),
			collections.ProtoValueEncoder[types.ReserveSnapshot](cdc),
		),
		DnREpoch: collections.NewItem(
			storeKey, NamespaceDnrEpoch,
			collections.Uint64ValueEncoder,
		),
		GlobalVolumes: collections.NewMap(
			storeKey, NamespaceGlobalVolumes,
			collections.Uint64KeyEncoder,
			collections.IntValueEncoder,
		),
		TraderVolumes: collections.NewMap(
			storeKey, NamespaceUserVolumes,
			collections.PairKeyEncoder(collections.AccAddressKeyEncoder, collections.Uint64KeyEncoder),
			collections.IntValueEncoder,
		),
		GlobalDiscounts: collections.NewMap(
			storeKey, NamespaceGlobalDiscounts,
			collections.IntKeyEncoder,
			collections.DecValueEncoder,
		),
		TraderDiscounts: collections.NewMap(
			storeKey, NamespaceUserDiscounts,
			collections.PairKeyEncoder(collections.AccAddressKeyEncoder, collections.IntKeyEncoder),
			collections.DecValueEncoder,
		),
		EpochRebateAllocations: collections.NewMap(
			storeKey, NamespaceRebatesAllocations,
			collections.Uint64KeyEncoder,
			collections.ProtoValueEncoder[types.DNRAllocation](cdc),
		),
		Collateral: collections.NewItem(
			storeKey, NamespaceCollateral,
			common.StringValueEncoder,
		),
		DnREpochName: collections.NewItem(
			storeKey, NamespaceDnrEpochName,
			common.StringValueEncoder,
		),
	}
	k.Admin = admin{&k}
	return k
}

const (
	NamespaceMarkets collections.Namespace = iota + 11 // == 11 because iota starts from 0
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

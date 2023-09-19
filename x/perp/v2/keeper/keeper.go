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

	Markets          collections.Map[asset.Pair, types.Market]
	AMMs             collections.Map[asset.Pair, types.AMM]
	Positions        collections.Map[collections.Pair[asset.Pair, sdk.AccAddress], types.Position]
	ReserveSnapshots collections.Map[collections.Pair[asset.Pair, time.Time], types.ReserveSnapshot]
	DnREpoch         collections.Item[uint64]
	TraderVolumes    collections.Map[collections.Pair[sdk.AccAddress, uint64], math.Int]         // Keeps track of user volumes for each epoch.
	GlobalDiscounts  collections.Map[math.Int, math.LegacyDec]                                   // maps a volume level to a discount
	TraderDiscounts  collections.Map[collections.Pair[sdk.AccAddress, math.Int], math.LegacyDec] // maps a user and volume level to a discount, supersedes global discounts
}

// NewKeeper Creates a new x/perp Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	oracleKeeper types.OracleKeeper,
	epochKeeper types.EpochKeeper,
) Keeper {
	// Ensure that the module account is set.
	if moduleAcc := accountKeeper.GetModuleAddress(types.ModuleName); moduleAcc == nil {
		panic("The x/perp module account has not been set")
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		BankKeeper:    bankKeeper,
		AccountKeeper: accountKeeper,
		OracleKeeper:  oracleKeeper,
		EpochKeeper:   epochKeeper,
		Markets: collections.NewMap(
			storeKey, NamespaceMarkets,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[types.Market](cdc),
		),
		AMMs: collections.NewMap(
			storeKey, NamespaceAmms,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[types.AMM](cdc),
		),
		Positions: collections.NewMap(
			storeKey, NamespacePositions,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.AccAddressKeyEncoder),
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
		TraderVolumes: collections.NewMap(
			storeKey, NamespaceUserVolumes,
			collections.PairKeyEncoder(collections.AccAddressKeyEncoder, collections.Uint64KeyEncoder),
			IntValueEncoder,
		),
		GlobalDiscounts: collections.NewMap(
			storeKey, NamespaceGlobalDiscounts,
			IntKeyEncoder,
			collections.DecValueEncoder,
		),
		TraderDiscounts: collections.NewMap(
			storeKey, NamespaceUserDiscounts,
			collections.PairKeyEncoder(collections.AccAddressKeyEncoder, IntKeyEncoder),
			collections.DecValueEncoder,
		),
	}
}

const (
	NamespaceMarkets collections.Namespace = iota + 11 // == 11 because iota starts from 0
	NamespaceAmms
	NamespacePositions
	NamespaceReserveSnapshots
	NamespaceDnrEpoch
	NamespaceUserVolumes
	NamespaceGlobalDiscounts
	NamespaceUserDiscounts
)

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// ChangeMarketEnabledParameter change the market enabled parameter
func (k Keeper) ChangeMarketEnabledParameter(ctx sdk.Context, pair asset.Pair, enabled bool) (err error) {
	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return
	}
	market.Enabled = enabled
	k.Markets.Insert(ctx, pair, market)
	return
}

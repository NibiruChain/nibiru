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

	MarketLastVersion collections.Map[asset.Pair, types.MarketLastVersion]
	Markets           collections.Map[collections.Pair[asset.Pair, uint64], types.Market]
	AMMs              collections.Map[collections.Pair[asset.Pair, uint64], types.AMM]

	Positions        collections.Map[collections.Pair[asset.Pair, sdk.AccAddress], types.Position]
	ReserveSnapshots collections.Map[collections.Pair[asset.Pair, time.Time], types.ReserveSnapshot]
	DnREpoch         collections.Item[uint64]
	TraderVolumes    collections.Map[collections.Pair[sdk.AccAddress, uint64], math.Int] // Keeps track of user volumes for each epoch.
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
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.Uint64KeyEncoder),
			collections.ProtoValueEncoder[types.Market](cdc),
		),
		MarketLastVersion: collections.NewMap(
			storeKey, NamespaceMarketLastVersion,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[types.MarketLastVersion](cdc),
		),
		AMMs: collections.NewMap(
			storeKey, NamespaceAmms,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.Uint64KeyEncoder),
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
	}
}

const (
	NamespaceMarkets collections.Namespace = iota + 11 // == 11 because iota starts from 0
	NamespaceAmms
	NamespacePositions
	NamespaceReserveSnapshots
	NamespaceDnrEpoch
	NamespaceUserVolumes
	NamespaceMarketLastVersion
)

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// ChangeMarketEnabledParameter change the market enabled parameter
func (k Keeper) ChangeMarketEnabledParameter(ctx sdk.Context, pair asset.Pair, enabled bool) (err error) {
	market, err := k.GetMarket(ctx, pair)
	if err != nil {
		return
	}
	market.Enabled = enabled
	k.SaveMarket(ctx, market)
	return
}

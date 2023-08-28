package keeper

import (
	"fmt"
	"time"

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

	Markets           collections.Map[collections.Pair[asset.Pair, uint64], types.Market]
	MarketLastVersion collections.Map[asset.Pair, types.MarketLastVersion]
	AMMs              collections.Map[asset.Pair, types.AMM]
	Positions         collections.Map[collections.Pair[asset.Pair, sdk.AccAddress], types.Position]
	ReserveSnapshots  collections.Map[collections.Pair[asset.Pair, time.Time], types.ReserveSnapshot]
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
			storeKey, 11,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.Uint64KeyEncoder),
			collections.ProtoValueEncoder[types.Market](cdc),
		),
		MarketLastVersion: collections.NewMap(
			storeKey, 15,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[types.MarketLastVersion](cdc),
		),
		AMMs: collections.NewMap(
			storeKey, 12,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[types.AMM](cdc),
		),
		Positions: collections.NewMap(
			storeKey, 13,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.AccAddressKeyEncoder),
			collections.ProtoValueEncoder[types.Position](cdc),
		),
		ReserveSnapshots: collections.NewMap(
			storeKey, 14,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.TimeKeyEncoder),
			collections.ProtoValueEncoder[types.ReserveSnapshot](cdc),
		),
	}
}

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
	k.Markets.Insert(ctx, collections.Join(pair, market.Version), market)
	return
}

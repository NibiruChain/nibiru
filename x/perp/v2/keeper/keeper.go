package keeper

import (
	"fmt"
	"time"

	"github.com/NibiruChain/collections"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/common/asset"
	v2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey sdk.StoreKey

	BankKeeper    v2types.BankKeeper
	AccountKeeper v2types.AccountKeeper
	OracleKeeper  v2types.OracleKeeper
	EpochKeeper   v2types.EpochKeeper

	Markets          collections.Map[asset.Pair, v2types.Market]
	AMMs             collections.Map[asset.Pair, v2types.AMM]
	Positions        collections.Map[collections.Pair[asset.Pair, sdk.AccAddress], v2types.Position]
	ReserveSnapshots collections.Map[collections.Pair[asset.Pair, time.Time], v2types.ReserveSnapshot]
}

// NewKeeper Creates a new x/perp Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,

	accountKeeper v2types.AccountKeeper,
	bankKeeper v2types.BankKeeper,
	oracleKeeper v2types.OracleKeeper,
	epochKeeper v2types.EpochKeeper,
) Keeper {
	// Ensure that the module account is set.
	if moduleAcc := accountKeeper.GetModuleAddress(v2types.ModuleName); moduleAcc == nil {
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
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[v2types.Market](cdc),
		),
		AMMs: collections.NewMap(
			storeKey, 12,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[v2types.AMM](cdc),
		),
		Positions: collections.NewMap(
			storeKey, 13,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.AccAddressKeyEncoder),
			collections.ProtoValueEncoder[v2types.Position](cdc),
		),
		ReserveSnapshots: collections.NewMap(
			storeKey, 14,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.TimeKeyEncoder),
			collections.ProtoValueEncoder[v2types.ReserveSnapshot](cdc),
		),
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", v2types.ModuleName))
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

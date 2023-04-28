package keeper

import (
	"fmt"
	"time"

	"github.com/NibiruChain/collections"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      sdk.StoreKey
	ParamSubspace paramtypes.Subspace

	BankKeeper    types.BankKeeper
	AccountKeeper types.AccountKeeper
	OracleKeeper  types.OracleKeeper
	EpochKeeper   types.EpochKeeper

	Markets          collections.Map[asset.Pair, v2types.Market]
	AMMs             collections.Map[asset.Pair, v2types.AMM]
	Positions        collections.Map[collections.Pair[asset.Pair, sdk.AccAddress], v2types.Position]
	Metrics          collections.Map[asset.Pair, v2types.Metrics]
	ReserveSnapshots collections.Map[collections.Pair[asset.Pair, time.Time], v2types.ReserveSnapshot]
}

// NewKeeper Creates a new x/perp Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	paramSubspace paramtypes.Subspace,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	oracleKeeper types.OracleKeeper,
	epochKeeper types.EpochKeeper,
) Keeper {
	// Ensure that the module account is set.
	if moduleAcc := accountKeeper.GetModuleAddress(types.ModuleName); moduleAcc == nil {
		panic("The x/perp module account has not been set")
	}

	// Set param.types.'KeyTable' if it has not already been set
	if !paramSubspace.HasKeyTable() {
		paramSubspace = paramSubspace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		ParamSubspace: paramSubspace,
		BankKeeper:    bankKeeper,
		AccountKeeper: accountKeeper,
		OracleKeeper:  oracleKeeper,
		EpochKeeper:   epochKeeper,
		Markets: collections.NewMap(
			storeKey, 0,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[v2types.Market](cdc),
		),
		AMMs: collections.NewMap(
			storeKey, 1,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[v2types.AMM](cdc),
		),
		Positions: collections.NewMap(
			storeKey, 2,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.AccAddressKeyEncoder),
			collections.ProtoValueEncoder[v2types.Position](cdc),
		),
		Metrics: collections.NewMap(storeKey, 3, asset.PairKeyEncoder, collections.ProtoValueEncoder[v2types.Metrics](cdc)),
		ReserveSnapshots: collections.NewMap(
			storeKey, 4,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.TimeKeyEncoder),
			collections.ProtoValueEncoder[v2types.ReserveSnapshot](cdc),
		),
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", v2types.ModuleName))
}

// GetParams get all parameters as v2types.Params
func (k Keeper) GetParams(ctx sdk.Context) (params v2types.Params) {
	k.ParamSubspace.GetParamSet(ctx, &params)
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params v2types.Params) {
	k.ParamSubspace.SetParamSet(ctx, &params)
}

// checkUserLimits checks if the limit is violated by the amount.
// returns error if it does
func checkUserLimits(limit, amount sdk.Dec, dir v2types.Direction) error {
	if limit.IsZero() {
		return nil
	}

	if dir == v2types.Direction_LONG && amount.LT(limit) {
		return v2types.ErrAssetFailsUserLimit.Wrapf(
			"amount (%s) is less than selected limit (%s)",
			amount.String(),
			limit.String(),
		)
	}

	if dir == v2types.Direction_SHORT && amount.GT(limit) {
		return v2types.ErrAssetFailsUserLimit.Wrapf(
			"amount (%s) is greater than selected limit (%s)",
			amount.String(),
			limit.String(),
		)
	}

	return nil
}

/*
*
Check's that a pool that we're about to save to state does not violate the fluctuation limit.
Always tries to check against a snapshot from a previous block. If one doesn't exist, then it just uses the current snapshot.
This should run prior to updating the snapshot, otherwise it will compare the currently updated market to itself.

args:
  - ctx: the cosmos-sdk context
  - pool: the updated market

ret:
  - err: error if any
*/
func (k Keeper) checkPriceFluctuationLimitRatio(ctx sdk.Context, market v2types.Market, amm v2types.AMM) error {
	if market.PriceFluctuationLimitRatio.IsZero() {
		// early return to avoid expensive state operations
		return nil
	}

	lastSnapshot, err := k.GetLastSnapshot(ctx, market.Pair)
	if err != nil {
		return err
	}
	if market.IsOverFluctuationLimitInRelationWithSnapshot(amm, lastSnapshot) {
		return v2types.ErrOverFluctuationLimit
	}

	return nil
}

/*
GetLastSnapshot retrieve the last snapshot for a particular market

args:
  - ctx: the cosmos-sdk context
  - pool: the market to check
*/
func (k Keeper) GetLastSnapshot(ctx sdk.Context, pair asset.Pair) (v2types.ReserveSnapshot, error) {
	it := k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}.Prefix(pair).Descending())
	defer it.Close()

	if !it.Valid() {
		return v2types.ReserveSnapshot{}, fmt.Errorf("error getting last snapshot number for pair %s", pair)
	}

	return it.Value(), nil
}

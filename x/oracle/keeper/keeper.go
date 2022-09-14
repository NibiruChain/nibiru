package keeper

import (
	"fmt"
	"github.com/NibiruChain/nibiru/collections"
	"github.com/NibiruChain/nibiru/collections/keys"

	"github.com/tendermint/tendermint/libs/log"

	gogotypes "github.com/gogo/protobuf/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// Keeper of the oracle store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   sdk.StoreKey
	paramSpace paramstypes.Subspace

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	distrKeeper   types.DistributionKeeper
	StakingKeeper types.StakingKeeper
	distrName     string

	Prevotes          collections.Map[keys.StringKey, types.AggregateExchangeRatePrevote, *types.AggregateExchangeRatePrevote]
	ExchangeRates     collections.Map[keys.StringKey, sdk.DecProto, *sdk.DecProto] // TODO: KEY is AssetPair, after AssetPair refactor.
	FeederDelegations collections.Map[keys.StringKey, gogotypes.BytesValue, *gogotypes.BytesValue]
	MissCounters      collections.Map[keys.StringKey, gogotypes.UInt64Value, *gogotypes.UInt64Value]
	PairRewardsID     collections.Sequence
}

// NewKeeper constructs a new keeper for oracle
func NewKeeper(cdc codec.BinaryCodec, storeKey sdk.StoreKey,
	paramspace paramstypes.Subspace, accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper, distrKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper, distrName string) Keeper {
	// ensure oracle module account is set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	// set KeyTable if it has not already been set
	if !paramspace.HasKeyTable() {
		paramspace = paramspace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:               cdc,
		storeKey:          storeKey,
		paramSpace:        paramspace,
		accountKeeper:     accountKeeper,
		bankKeeper:        bankKeeper,
		distrKeeper:       distrKeeper,
		StakingKeeper:     stakingKeeper,
		distrName:         distrName,
		Prevotes:          collections.NewMap[keys.StringKey, types.AggregateExchangeRatePrevote](cdc, storeKey, 0),
		ExchangeRates:     collections.NewMap[keys.StringKey, sdk.DecProto](cdc, storeKey, 1),
		FeederDelegations: collections.NewMap[keys.StringKey, gogotypes.BytesValue](cdc, storeKey, 2),
		MissCounters:      collections.NewMap[keys.StringKey, gogotypes.UInt64Value](cdc, storeKey, 3),
		PairRewardsID:     collections.NewSequence(cdc, storeKey, 6),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

//-----------------------------------
// ExchangeRate logic

// SetExchangeRateWithEvent calls SetExchangeRate and emits an event.
func (k Keeper) SetExchangeRateWithEvent(ctx sdk.Context, pair string, exchangeRate sdk.Dec) {
	k.ExchangeRates.Insert(ctx, keys.String(pair), sdk.DecProto{Dec: exchangeRate})
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(types.EventTypeExchangeRateUpdate,
			sdk.NewAttribute(types.AttributeKeyPair, pair),
			sdk.NewAttribute(types.AttributeKeyExchangeRate, exchangeRate.String()),
		),
	)
}

// AggregateExchangeRateVote logic

// GetAggregateExchangeRateVote retrieves an oracle prevote from the store
func (k Keeper) GetAggregateExchangeRateVote(ctx sdk.Context, voter sdk.ValAddress) (aggregateVote types.AggregateExchangeRateVote, err error) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.GetAggregateExchangeRateVoteKey(voter))
	if b == nil {
		err = sdkerrors.Wrap(types.ErrNoAggregateVote, voter.String())
		return
	}
	k.cdc.MustUnmarshal(b, &aggregateVote)
	return
}

// SetAggregateExchangeRateVote adds an oracle aggregate prevote to the store
func (k Keeper) SetAggregateExchangeRateVote(ctx sdk.Context, voter sdk.ValAddress, vote types.AggregateExchangeRateVote) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&vote)
	store.Set(types.GetAggregateExchangeRateVoteKey(voter), bz)
}

// DeleteAggregateExchangeRateVote deletes an oracle prevote from the store
func (k Keeper) DeleteAggregateExchangeRateVote(ctx sdk.Context, voter sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetAggregateExchangeRateVoteKey(voter))
}

// IterateAggregateExchangeRateVotes iterates rate over prevotes in the store
func (k Keeper) IterateAggregateExchangeRateVotes(ctx sdk.Context, handler func(voterAddr sdk.ValAddress, aggregateVote types.AggregateExchangeRateVote) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iter := sdk.KVStorePrefixIterator(store, types.AggregateExchangeRateVoteKey)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		voterAddr := sdk.ValAddress(iter.Key()[2:])

		var aggregateVote types.AggregateExchangeRateVote
		k.cdc.MustUnmarshal(iter.Value(), &aggregateVote)
		if handler(voterAddr, aggregateVote) {
			break
		}
	}
}

// PairExists return tobin tax for the pair
// TODO(mercilex): use AssetPair
func (k Keeper) PairExists(ctx sdk.Context, pair string) bool {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetPairKey(pair))
	return bz != nil
}

// SetPair updates tobin tax for the pair
// TODO(mercilex): use AssetPair
func (k Keeper) SetPair(ctx sdk.Context, pair string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetPairKey(pair), []byte{})
}

// IteratePairs iterates rate over tobin taxes in the store
func (k Keeper) IteratePairs(ctx sdk.Context, handler func(pair string) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iter := sdk.KVStorePrefixIterator(store, types.PairsKey)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		pair := types.ExtractPairFromPairKey(iter.Key())
		if handler(pair) {
			break
		}
	}
}

// ClearPairs clears tobin taxes
func (k Keeper) ClearPairs(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	iter := sdk.KVStorePrefixIterator(store, types.PairsKey)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		store.Delete(iter.Key())
	}
}

// ValidateFeeder return the given feeder is allowed to feed the message or not
func (k Keeper) ValidateFeeder(ctx sdk.Context, feederAddr sdk.AccAddress, validatorAddr sdk.ValAddress) error {
	if !feederAddr.Equals(validatorAddr) {
		// delegation by default is the one of the validator acc address
		delegate := sdk.AccAddress(k.FeederDelegations.GetOr(ctx, keys.String(validatorAddr.String()), gogotypes.BytesValue{Value: validatorAddr}).Value)
		if !delegate.Equals(feederAddr) {
			return sdkerrors.Wrapf(types.ErrNoVotingPermission, "wanted: %s, got: %s", delegate.String(), feederAddr.String())
		}
	}

	// Check that the given validator exists
	if val := k.StakingKeeper.Validator(ctx, validatorAddr); val == nil || !val.IsBonded() {
		return sdkerrors.Wrapf(stakingtypes.ErrNoValidatorFound, "validator %s is not active set", validatorAddr.String())
	}

	return nil
}

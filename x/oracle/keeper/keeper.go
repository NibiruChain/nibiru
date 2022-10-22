package keeper

import (
	"fmt"

	"github.com/NibiruChain/nibiru/collections"

	"github.com/tendermint/tendermint/libs/log"

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

	distrName string

	// TODO(mercilex): use asset pair
	ExchangeRates     collections.Map[string, sdk.Dec]
	FeederDelegations collections.Map[sdk.ValAddress, sdk.AccAddress]
	MissCounters      collections.Map[sdk.ValAddress, uint64]
	Prevotes          collections.Map[sdk.ValAddress, types.AggregateExchangeRatePrevote]
	Votes             collections.Map[sdk.ValAddress, types.AggregateExchangeRateVote]
	// TODO(mercilex): use asset pair
	Pairs         collections.KeySet[string]
	PairRewards   collections.IndexedMap[uint64, types.PairReward, PairRewardsIndexes]
	PairRewardsID collections.Sequence
}

type PairRewardsIndexes struct {
	// RewardsByPair is the index that maps rewards associated with specific pairs.
	RewardsByPair collections.MultiIndex[string, uint64, types.PairReward]
}

func (p PairRewardsIndexes) IndexerList() []collections.Indexer[uint64, types.PairReward] {
	return []collections.Indexer[uint64, types.PairReward]{p.RewardsByPair}
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
		ExchangeRates:     collections.NewMap(storeKey, 1, collections.StringKeyEncoder, collections.DecValueEncoder),
		FeederDelegations: collections.NewMap(storeKey, 2, collections.ValAddressKeyEncoder, collections.AccAddressValueEncoder),
		MissCounters:      collections.NewMap(storeKey, 3, collections.ValAddressKeyEncoder, collections.Uint64ValueEncoder),
		Prevotes:          collections.NewMap(storeKey, 4, collections.ValAddressKeyEncoder, collections.ProtoValueEncoder[types.AggregateExchangeRatePrevote](cdc)),
		Votes:             collections.NewMap(storeKey, 5, collections.ValAddressKeyEncoder, collections.ProtoValueEncoder[types.AggregateExchangeRateVote](cdc)),
		Pairs:             collections.NewKeySet(storeKey, 6, collections.StringKeyEncoder),
		PairRewards: collections.NewIndexedMap(
			storeKey, 7,
			collections.Uint64KeyEncoder, collections.ProtoValueEncoder[types.PairReward](cdc),
			PairRewardsIndexes{
				RewardsByPair: collections.NewMultiIndex(storeKey, 8, collections.StringKeyEncoder, collections.Uint64KeyEncoder, func(v types.PairReward) string {
					return v.Pair
				}),
			}),
		PairRewardsID: collections.NewSequence(storeKey, 9),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// ValidateFeeder return the given feeder is allowed to feed the message or not
func (k Keeper) ValidateFeeder(ctx sdk.Context, feederAddr sdk.AccAddress, validatorAddr sdk.ValAddress) error {
	if !feederAddr.Equals(validatorAddr) {
		delegate := k.FeederDelegations.GetOr(ctx, validatorAddr, sdk.AccAddress(validatorAddr)) // the right is delegated to himself by default
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

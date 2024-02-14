package keeper

import (
	"github.com/NibiruChain/collections"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/NibiruChain/nibiru/x/inflation/types"
)

// Keeper of the inflation store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	paramSpace paramstypes.Subspace

	// the address capable of executing a MsgUpdateParams message. Typically, this should be the x/gov module account.
	accountKeeper    types.AccountKeeper
	bankKeeper       types.BankKeeper
	distrKeeper      types.DistrKeeper
	stakingKeeper    types.StakingKeeper
	sudoKeeper       types.SudoKeeper
	epochsKeeper     types.EpochsKeeper
	feeCollectorName string

	CurrentPeriod    collections.Sequence
	NumSkippedEpochs collections.Sequence
	Params           collections.Item[types.Params]
}

// NewKeeper creates a new mint Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	paramspace paramstypes.Subspace,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distributionKeeper types.DistrKeeper,
	stakingKeeper types.StakingKeeper,
	sudoKeeper types.SudoKeeper,
	feeCollectorName string,
	epochsKeeper types.EpochsKeeper,
) Keeper {
	// ensure mint module account is set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the inflation module account has not been set")
	}

	return Keeper{
		storeKey:         storeKey,
		cdc:              cdc,
		paramSpace:       paramspace,
		accountKeeper:    accountKeeper,
		bankKeeper:       bankKeeper,
		distrKeeper:      distributionKeeper,
		stakingKeeper:    stakingKeeper,
		sudoKeeper:       sudoKeeper,
		epochsKeeper:     epochsKeeper,
		feeCollectorName: feeCollectorName,
		CurrentPeriod:    collections.NewSequence(storeKey, 0),
		NumSkippedEpochs: collections.NewSequence(storeKey, 1),
		Params:           collections.NewItem(storeKey, 2, collections.ProtoValueEncoder[types.Params](cdc)),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

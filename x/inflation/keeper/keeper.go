package keeper

import (
	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tendermint/tendermint/libs/log"

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
	feeCollectorName string

	CurrentPeriod    collections.Sequence
	NumSkippedEpochs collections.Sequence
}

// NewKeeper creates a new mint Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	paramspace paramstypes.Subspace,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	dk types.DistrKeeper,
	sk types.StakingKeeper,
	feeCollectorName string,
) Keeper {
	// ensure mint module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the inflation module account has not been set")
	}

	// set KeyTable if it has not already been set
	if !paramspace.HasKeyTable() {
		paramspace = paramspace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		storeKey:         storeKey,
		cdc:              cdc,
		paramSpace:       paramspace,
		accountKeeper:    ak,
		bankKeeper:       bk,
		distrKeeper:      dk,
		stakingKeeper:    sk,
		feeCollectorName: feeCollectorName,
		CurrentPeriod:    collections.NewSequence(storeKey, 0),
		NumSkippedEpochs: collections.NewSequence(storeKey, 1),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

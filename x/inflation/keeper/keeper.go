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

// Keeper of the inflation module. Keepers are module-specific "gate keepers"
// responsible for encapsulating access to the key-value stores (state) of the
// network. The functions on the Keeper contain all the business logic for
// reading and modifying state.
type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	// paramSpace: unused but present for backward compatibility. Removing this
	// breaks the state machine and requires an upgrade.
	paramSpace paramstypes.Subspace

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	distrKeeper   types.DistrKeeper
	stakingKeeper types.StakingKeeper
	sudoKeeper    types.SudoKeeper
	// feeCollectorName is the name of x/auth module's fee collector module
	// account, "fee_collector", which collects transaction fees for distribution
	// to all stakers.
	// By sending staking inflation to the fee collector, the tokens are properly
	// distributed to validator operators and their delegates.
	// See the `[AllocateTokens]` function from x/distribution to learn more.
	// [AllocateTokens]: https://github.com/cosmos/cosmos-sdk/blob/v0.50.3/x/distribution/keeper/allocation.go
	feeCollectorName string

	// CurrentPeriod: Strictly increasing counter for the inflation "period".
	CurrentPeriod collections.Sequence

	// NumSkippedEpochs: Strictly increasing counter for the number of skipped
	// epochs. Inflation epochs are skipped when [types.Params.InflationEnabled]
	// is false so that gaps in the active status of inflation don't mess up the
	// polynomial computation. It allows inflation to smoothly be toggled on and
	// off.
	NumSkippedEpochs collections.Sequence

	// Params stores module-specific parameters that specify the blockchain token
	// economics, token release schedule, maximum supply, and whether or not
	// inflation is enabled on the network.
	Params collections.Item[types.Params]
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

func (k Keeper) Burn(ctx sdk.Context, coins sdk.Coins, sender sdk.AccAddress) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, sender, types.ModuleName, coins,
	); err != nil {
		return err
	}

	return k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
}

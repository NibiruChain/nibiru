package keeper

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	storetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/cometbft/cometbft/libs/log"

	"github.com/NibiruChain/nibiru/v2/x/collections"

	"github.com/NibiruChain/nibiru/v2/x/mint"
)

// Keeper of the inflation module. Keepers are module-specific "gate keepers"
// responsible for encapsulating access to the key-value stores (state) of the
// network. The functions on the Keeper contain all the business logic for
// reading and modifying state.
type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	accountKeeper mint.AccountKeeper
	bankKeeper    mint.BankKeeper
	distrKeeper   mint.DistrKeeper
	stakingKeeper mint.StakingKeeper
	sudoKeeper    mint.SudoKeeper
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
	Params collections.Item[mint.Params]
}

// NewKeeper creates a new mint Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	accountKeeper mint.AccountKeeper,
	bankKeeper mint.BankKeeper,
	distributionKeeper mint.DistrKeeper,
	stakingKeeper mint.StakingKeeper,
	sudoKeeper mint.SudoKeeper,
	feeCollectorName string,
) Keeper {
	// ensure mint module account is set
	if addr := accountKeeper.GetModuleAddress(mint.ModuleName); addr == nil {
		panic("the inflation module account has not been set")
	}

	return Keeper{
		storeKey:         storeKey,
		cdc:              cdc,
		accountKeeper:    accountKeeper,
		bankKeeper:       bankKeeper,
		distrKeeper:      distributionKeeper,
		stakingKeeper:    stakingKeeper,
		sudoKeeper:       sudoKeeper,
		feeCollectorName: feeCollectorName,
		CurrentPeriod:    collections.NewSequence(storeKey, 0),
		NumSkippedEpochs: collections.NewSequence(storeKey, 1),
		Params:           collections.NewItem(storeKey, 2, collections.ProtoValueEncoder[mint.Params](cdc)),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+mint.ModuleName)
}

func (k Keeper) Burn(ctx sdk.Context, coins sdk.Coins, sender sdk.AccAddress) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, sender, mint.ModuleName, coins,
	); err != nil {
		return err
	}

	return k.bankKeeper.BurnCoins(ctx, mint.ModuleName, coins)
}

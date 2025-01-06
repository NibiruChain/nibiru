package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

var _ bankkeeper.Keeper = &NibiruBankKeeper{}

type NibiruBankKeeper struct {
	bankkeeper.BaseKeeper
	StateDB *statedb.StateDB
}

func (evmKeeper *Keeper) NewStateDB(
	ctx sdk.Context, txConfig statedb.TxConfig,
) *statedb.StateDB {
	stateDB := statedb.New(ctx, evmKeeper, txConfig)
	evmKeeper.Bank.StateDB = stateDB
	return stateDB
}

func (bk NibiruBankKeeper) InputOutputCoins(
	ctx sdk.Context,
	input []banktypes.Input,
	output []banktypes.Output,
) error {
	return bk.ForceGasInvariant(
		ctx,
		func(ctx sdk.Context) error {
			return bk.BaseKeeper.InputOutputCoins(ctx, input, output)
		},
		func(ctx sdk.Context) {
			for _, input := range input {
				if findEtherBalanceChangeFromCoins(input.Coins) {
					bk.SyncStateDBWithAccount(ctx, sdk.MustAccAddressFromBech32(input.Address))
				}
			}
			for _, output := range output {
				if findEtherBalanceChangeFromCoins(output.Coins) {
					bk.SyncStateDBWithAccount(ctx, sdk.MustAccAddressFromBech32(output.Address))
				}
			}
		},
	)
}

func (bk NibiruBankKeeper) DelegateCoins(
	ctx sdk.Context,
	delegatorAddr sdk.AccAddress,
	moduleBech32Addr sdk.AccAddress,
	coins sdk.Coins,
) error {
	return bk.ForceGasInvariant(
		ctx,
		func(ctx sdk.Context) error {
			return bk.BaseKeeper.DelegateCoins(ctx, delegatorAddr, moduleBech32Addr, coins)
		},
		func(ctx sdk.Context) {
			if findEtherBalanceChangeFromCoins(coins) {
				bk.SyncStateDBWithAccount(ctx, delegatorAddr)
			}
		},
	)
}

func (bk NibiruBankKeeper) UndelegateCoins(
	ctx sdk.Context,
	delegatorAddr sdk.AccAddress,
	moduleBech32Addr sdk.AccAddress,
	coins sdk.Coins,
) error {
	return bk.ForceGasInvariant(
		ctx,
		func(ctx sdk.Context) error {
			return bk.BaseKeeper.UndelegateCoins(ctx, delegatorAddr, moduleBech32Addr, coins)
		},
		func(ctx sdk.Context) {
			if findEtherBalanceChangeFromCoins(coins) {
				bk.SyncStateDBWithAccount(ctx, delegatorAddr)
			}
		},
	)
}

func (bk NibiruBankKeeper) MintCoins(
	ctx sdk.Context,
	moduleName string,
	coins sdk.Coins,
) error {
	return bk.ForceGasInvariant(
		ctx,
		func(ctx sdk.Context) error {
			// Use the embedded function from [bankkeeper.Keeper]
			return bk.BaseKeeper.MintCoins(ctx, moduleName, coins)
		},
		func(ctx sdk.Context) {
			if findEtherBalanceChangeFromCoins(coins) {
				moduleBech32Addr := auth.NewModuleAddress(moduleName)
				bk.SyncStateDBWithAccount(ctx, moduleBech32Addr)
			}
		},
	)
}

func (bk NibiruBankKeeper) BurnCoins(
	ctx sdk.Context,
	moduleName string,
	coins sdk.Coins,
) error {
	return bk.ForceGasInvariant(
		ctx,
		func(ctx sdk.Context) error {
			// Use the embedded function from [bankkeeper.Keeper]
			return bk.BaseKeeper.BurnCoins(ctx, moduleName, coins)
		},
		func(ctx sdk.Context) {
			if findEtherBalanceChangeFromCoins(coins) {
				moduleBech32Addr := auth.NewModuleAddress(moduleName)
				bk.SyncStateDBWithAccount(ctx, moduleBech32Addr)
			}
		},
	)
}

// Each Send* operation on the [NibiruBankKeeper] can be described as having a
// base operation (BaseOp) where the [bankkeeper.BaseKeeper] executes some
// business logic and an operation that occurs afterward (AfterOp), where we
// post-process and provide automatic alignment with the EVM [statedb.StateDB].
//
// Each "AfterOp" tends to consume a negligible amount of gas (<2000 gas), while
// a each "BaseOp" is around 27000 for a single coin transfer.
//
// Although each "AfterOp" consumes a negligible amount of gas, that
// amount of gas consumed is nonzero and depends on whether the current bank
// transaction message occurs within an Ethereum tx or not.
//
// Consistent gas consumption independent of status of the EVM StateDB is brought
// about in [ForceGasInvariant] by consuming only the gas used for the BaseOp.
// This makes sure that post-processing for the EVM [statedb.StateDB] will not
// result in nondeterminism.
func (bk NibiruBankKeeper) ForceGasInvariant(
	ctx sdk.Context,
	BaseOp func(ctx sdk.Context) error,
	AfterOp func(ctx sdk.Context),
) error {
	// Assign vars for the tx gas meter
	gasMeterBefore := ctx.GasMeter() // Tx gas meter MUST be defined
	gasConsumedBefore := gasMeterBefore.GasConsumed()
	// Don't modify the "ctx.BlockGasMeter()" directly because this is
	// handled in "BaseApp.runTx"

	// Start baseGasConsumed at 0 in case we panic before BaseOp completes and
	// baseGasConsumed gets a value assignment
	baseOpGasConsumed := uint64(0)

	defer func() {
		gasMeterBefore.RefundGas(gasMeterBefore.GasConsumed(), "")
		gasMeterBefore.ConsumeGas(gasConsumedBefore+baseOpGasConsumed, "NibiruBankKeeper invariant")
	}()

	// Note that because the ctx gas meter uses private variables to track gas,
	// we have to branch off with a new gas meter instance to avoid mutating the
	// "true" gas meter (called GasMeterBefore here).
	// We use an infinite gas meter because we consume gas in the deferred function
	// and gasMeterBefore will panic if we consume too much gas.
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())

	err := BaseOp(ctx)
	baseOpGasConsumed = ctx.GasMeter().GasConsumed()
	if err != nil {
		return err
	}

	AfterOp(ctx)
	return nil
}

func (bk NibiruBankKeeper) SendCoins(
	ctx sdk.Context,
	fromAddr sdk.AccAddress,
	toAddr sdk.AccAddress,
	coins sdk.Coins,
) error {
	return bk.ForceGasInvariant(
		ctx,
		func(ctx sdk.Context) error {
			return bk.BaseKeeper.SendCoins(ctx, fromAddr, toAddr, coins)
		},
		func(ctx sdk.Context) {
			if findEtherBalanceChangeFromCoins(coins) {
				bk.SyncStateDBWithAccount(ctx, fromAddr)
				bk.SyncStateDBWithAccount(ctx, toAddr)
			}
		},
	)
}

func (bk *NibiruBankKeeper) SyncStateDBWithAccount(
	ctx sdk.Context, acc sdk.AccAddress,
) {
	// If there's no StateDB set, it means we're not in an EthereumTx.
	if bk.StateDB == nil {
		return
	}

	balanceWei := evm.NativeToWei(
		bk.GetBalance(ctx, acc, evm.EVMBankDenom).Amount.BigInt(),
	)
	bk.StateDB.SetBalanceWei(eth.NibiruAddrToEthAddr(acc), balanceWei)
}

func findEtherBalanceChangeFromCoins(coins sdk.Coins) (found bool) {
	for _, c := range coins {
		if c.Denom == evm.EVMBankDenom {
			return true
		}
	}
	return false
}

func (bk NibiruBankKeeper) SendCoinsFromAccountToModule(
	ctx sdk.Context,
	senderAddr sdk.AccAddress,
	recipientModule string,
	coins sdk.Coins,
) error {
	return bk.ForceGasInvariant(
		ctx,
		func(ctx sdk.Context) error {
			// Use the embedded function from [bankkeeper.Keeper]
			return bk.BaseKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, coins)
		},
		func(ctx sdk.Context) {
			if findEtherBalanceChangeFromCoins(coins) {
				bk.SyncStateDBWithAccount(ctx, senderAddr)
				moduleBech32Addr := auth.NewModuleAddress(recipientModule)
				bk.SyncStateDBWithAccount(ctx, moduleBech32Addr)
			}
		},
	)
}

func (bk NibiruBankKeeper) SendCoinsFromModuleToAccount(
	ctx sdk.Context,
	senderModule string,
	recipientAddr sdk.AccAddress,
	coins sdk.Coins,
) error {
	return bk.ForceGasInvariant(
		ctx,
		func(ctx sdk.Context) error {
			// Use the embedded function from [bankkeeper.Keeper]
			return bk.BaseKeeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, coins)
		},
		func(ctx sdk.Context) {
			if findEtherBalanceChangeFromCoins(coins) {
				moduleBech32Addr := auth.NewModuleAddress(senderModule)
				bk.SyncStateDBWithAccount(ctx, moduleBech32Addr)
				bk.SyncStateDBWithAccount(ctx, recipientAddr)
			}
		},
	)
}

func (bk NibiruBankKeeper) SendCoinsFromModuleToModule(
	ctx sdk.Context,
	senderModule string,
	recipientModule string,
	coins sdk.Coins,
) error {
	return bk.ForceGasInvariant(
		ctx,
		func(ctx sdk.Context) error {
			// Use the embedded function from [bankkeeper.Keeper]
			return bk.BaseKeeper.SendCoinsFromModuleToModule(ctx, senderModule, recipientModule, coins)
		},
		func(ctx sdk.Context) {
			if findEtherBalanceChangeFromCoins(coins) {
				senderBech32Addr := auth.NewModuleAddress(senderModule)
				recipientBech32Addr := auth.NewModuleAddress(recipientModule)
				bk.SyncStateDBWithAccount(ctx, senderBech32Addr)
				bk.SyncStateDBWithAccount(ctx, recipientBech32Addr)
			}
		},
	)
}

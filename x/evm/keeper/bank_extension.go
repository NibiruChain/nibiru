package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

var (
	_ bankkeeper.Keeper     = &NibiruBankKeeper{}
	_ bankkeeper.SendKeeper = &NibiruBankKeeper{}
)

type NibiruBankKeeper struct {
	bankkeeper.BaseKeeper
	StateDB   *statedb.StateDB
	TxStateDB *statedb.StateDB
}

func (evmKeeper *Keeper) NewStateDB(
	ctx sdk.Context, txConfig statedb.TxConfig,
) *statedb.StateDB {
	stateDB := statedb.New(ctx, evmKeeper, txConfig)
	evmKeeper.Bank.StateDB = stateDB
	return stateDB
}

func (evmKeeper *Keeper) NewTxStateDB(
	ctx sdk.Context, txConfig statedb.TxConfig,
) *statedb.StateDB {
	stateDB := statedb.New(ctx, evmKeeper, txConfig)
	evmKeeper.Bank.StateDB = stateDB
	evmKeeper.Bank.TxStateDB = stateDB
	return stateDB
}

func (bk NibiruBankKeeper) MintCoins(
	ctx sdk.Context,
	moduleName string,
	coins sdk.Coins,
) error {
	// Use the embedded function from [bankkeeper.Keeper]
	if err := bk.BaseKeeper.MintCoins(ctx, moduleName, coins); err != nil {
		return err
	}
	if findEtherBalanceChangeFromCoins(coins) {
		moduleBech32Addr := auth.NewModuleAddress(evm.ModuleName)
		bk.SyncStateDBWithAccount(ctx, moduleBech32Addr)
	}
	return nil
}

func (bk NibiruBankKeeper) BurnCoins(
	ctx sdk.Context,
	moduleName string,
	coins sdk.Coins,
) error {
	// Use the embedded function from [bankkeeper.Keeper]
	if err := bk.BaseKeeper.BurnCoins(ctx, moduleName, coins); err != nil {
		return err
	}
	if findEtherBalanceChangeFromCoins(coins) {
		moduleBech32Addr := auth.NewModuleAddress(evm.ModuleName)
		bk.SyncStateDBWithAccount(ctx, moduleBech32Addr)
	}
	return nil
}

func (bk NibiruBankKeeper) SendCoins(
	ctx sdk.Context,
	fromAddr sdk.AccAddress,
	toAddr sdk.AccAddress,
	coins sdk.Coins,
) error {
	// Use the embedded function from [bankkeeper.Keeper]
	if err := bk.BaseKeeper.SendCoins(ctx, fromAddr, toAddr, coins); err != nil {
		return err
	}
	if findEtherBalanceChangeFromCoins(coins) {
		bk.SyncStateDBWithAccount(ctx, fromAddr)
		bk.SyncStateDBWithAccount(ctx, toAddr)
	}
	return nil
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
	// Use the embedded function from [bankkeeper.Keeper]
	if err := bk.BaseKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, coins); err != nil {
		return err
	}
	if findEtherBalanceChangeFromCoins(coins) {
		bk.SyncStateDBWithAccount(ctx, senderAddr)
		moduleBech32Addr := auth.NewModuleAddress(recipientModule)
		bk.SyncStateDBWithAccount(ctx, moduleBech32Addr)
	}
	return nil
}

func (bk NibiruBankKeeper) SendCoinsFromModuleToAccount(
	ctx sdk.Context,
	senderModule string,
	recipientAddr sdk.AccAddress,
	coins sdk.Coins,
) error {
	// Use the embedded function from [bankkeeper.Keeper]
	if err := bk.BaseKeeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, coins); err != nil {
		return err
	}
	if findEtherBalanceChangeFromCoins(coins) {
		moduleBech32Addr := auth.NewModuleAddress(senderModule)
		bk.SyncStateDBWithAccount(ctx, moduleBech32Addr)
		bk.SyncStateDBWithAccount(ctx, recipientAddr)
	}
	return nil
}

func (bk NibiruBankKeeper) SendCoinsFromModuleToModule(
	ctx sdk.Context,
	senderModule string,
	recipientModule string,
	coins sdk.Coins,
) error {
	// Use the embedded function from [bankkeeper.Keeper]
	if err := bk.BaseKeeper.SendCoinsFromModuleToModule(ctx, senderModule, recipientModule, coins); err != nil {
		return err
	}
	if findEtherBalanceChangeFromCoins(coins) {
		senderBech32Addr := auth.NewModuleAddress(senderModule)
		recipientBech32Addr := auth.NewModuleAddress(recipientModule)
		bk.SyncStateDBWithAccount(ctx, senderBech32Addr)
		bk.SyncStateDBWithAccount(ctx, recipientBech32Addr)
	}
	return nil
}

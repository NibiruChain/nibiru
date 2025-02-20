// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

var _ statedb.Keeper = &Keeper{}

// ----------------------------------------------------------------------------
// StateDB Keeper implementation
// ----------------------------------------------------------------------------

// GetAccount: Ethereum account getter for a [statedb.Account].
// Implements the `statedb.Keeper` interface.
// Returns nil if the account does not not exist or has the wrong type.
func (k *Keeper) GetAccount(ctx sdk.Context, addr gethcommon.Address) *statedb.Account {
	acct := k.GetAccountWithoutBalance(ctx, addr)
	if acct == nil {
		return nil
	}

	acct.Balance = k.GetEvmGasBalance(ctx, addr)
	return acct
}

// GetCode: Loads smart contract bytecode.
// Implements the `statedb.Keeper` interface.
func (k *Keeper) GetCode(ctx sdk.Context, codeHash gethcommon.Hash) []byte {
	codeBz, err := k.EvmState.ContractBytecode.Get(ctx, codeHash.Bytes())
	if err != nil {
		panic(err) // TODO: We don't like to panic.
	}
	return codeBz
}

// ForEachStorage: Iterator over contract storage.
// Implements the `statedb.Keeper` interface.
func (k *Keeper) ForEachStorage(
	ctx sdk.Context,
	addr gethcommon.Address,
	stopIter func(key, value gethcommon.Hash) bool,
) {
	store := ctx.KVStore(k.storeKey)
	prefix := evm.PrefixAccStateEthAddr(addr)

	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := gethcommon.BytesToHash(iterator.Key())
		value := gethcommon.BytesToHash(iterator.Value())

		// check if iteration stops
		if !stopIter(key, value) {
			return
		}
	}
}

// SetAccBalance update account's balance, compare with current balance first, then decide to mint or burn.
func (k *Keeper) SetAccBalance(
	ctx sdk.Context, addr gethcommon.Address, amount *big.Int,
) error {
	nativeAddr := sdk.AccAddress(addr.Bytes())
	params := k.GetParams(ctx)
	coin := k.bankKeeper.GetBalance(ctx, nativeAddr, params.EvmDenom)
	balance := coin.Amount.BigInt()
	delta := new(big.Int).Sub(amount, balance)
	switch delta.Sign() {
	case 1:
		// mint
		coins := sdk.NewCoins(sdk.NewCoin(params.EvmDenom, sdkmath.NewIntFromBigInt(delta)))
		if err := k.bankKeeper.MintCoins(ctx, evm.ModuleName, coins); err != nil {
			return err
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, evm.ModuleName, nativeAddr, coins); err != nil {
			return err
		}
	case -1:
		// burn
		coins := sdk.NewCoins(sdk.NewCoin(params.EvmDenom, sdkmath.NewIntFromBigInt(new(big.Int).Neg(delta))))
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, nativeAddr, evm.ModuleName, coins); err != nil {
			return err
		}
		if err := k.bankKeeper.BurnCoins(ctx, evm.ModuleName, coins); err != nil {
			return err
		}
	default:
		// not changed
	}
	return nil
}

// SetAccount: Updates nonce, balance, and codeHash.
// Implements the `statedb.Keeper` interface.
// Only called by `StateDB.Commit()`.
func (k *Keeper) SetAccount(
	ctx sdk.Context, addr gethcommon.Address, account statedb.Account,
) error {
	// update account
	nibiruAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, nibiruAddr)
	if acct == nil {
		acct = k.accountKeeper.NewAccountWithAddress(ctx, nibiruAddr)
	}

	if err := acct.SetSequence(account.Nonce); err != nil {
		return err
	}

	codeHash := gethcommon.BytesToHash(account.CodeHash)

	if ethAcct, ok := acct.(eth.EthAccountI); ok {
		if err := ethAcct.SetCodeHash(codeHash); err != nil {
			return err
		}
	}

	k.accountKeeper.SetAccount(ctx, acct)

	if err := k.SetAccBalance(ctx, addr, account.Balance); err != nil {
		return err
	}

	return nil
}

// SetState:  Update contract storage, delete if value is empty.
// Implements the `statedb.Keeper` interface.
// Only called by `StateDB.Commit()`.
func (k *Keeper) SetState(
	ctx sdk.Context, addr gethcommon.Address, stateKey gethcommon.Hash, stateValue []byte,
) {
	k.EvmState.SetAccState(ctx, addr, stateKey, stateValue)
}

// SetCode: Setter for smart contract bytecode. Delete if code is empty.
// Implements the `statedb.Keeper` interface.
// Only called by `StateDB.Commit()`.
func (k *Keeper) SetCode(ctx sdk.Context, codeHash, code []byte) {
	k.EvmState.SetAccCode(ctx, codeHash, code)
}

// DeleteAccount handles contract's suicide call, clearing the balance, contract
// bytecode, contract state, and its native account.
// Implements the `statedb.Keeper` interface.
// Only called by `StateDB.Commit()`.
func (k *Keeper) DeleteAccount(ctx sdk.Context, addr gethcommon.Address) error {
	nibiruAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, nibiruAddr)
	if acct == nil {
		return nil
	}

	_, ok := acct.(eth.EthAccountI)
	if !ok {
		return evm.ErrInvalidAccount.Wrapf("type %T, address %s", acct, addr)
	}

	// clear balance
	if err := k.SetAccBalance(ctx, addr, new(big.Int)); err != nil {
		return err
	}

	// clear storage
	k.ForEachStorage(ctx, addr, func(key, _ gethcommon.Hash) bool {
		k.SetState(ctx, addr, key, nil)
		return true
	})

	k.accountKeeper.RemoveAccount(ctx, acct)

	return nil
}

// GetAccountWithoutBalance load nonce and codehash without balance,
// more efficient in cases where balance is not needed.
func (k *Keeper) GetAccountWithoutBalance(ctx sdk.Context, addr gethcommon.Address) *statedb.Account {
	nibiruAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, nibiruAddr)
	if acct == nil {
		return nil
	}

	codeHash := evm.EmptyCodeHash
	ethAcct, ok := acct.(eth.EthAccountI)
	if ok {
		codeHash = ethAcct.GetCodeHash().Bytes()
	}

	return &statedb.Account{
		Nonce:    acct.GetSequence(),
		CodeHash: codeHash,
	}
}

// GetAccountOrEmpty returns empty account if not exist, returns error if it's not `EthAccount`
func (k *Keeper) GetAccountOrEmpty(
	ctx sdk.Context, addr gethcommon.Address,
) statedb.Account {
	acct := k.GetAccount(ctx, addr)
	if acct != nil {
		return *acct
	}

	// empty account
	return statedb.Account{
		Balance:  new(big.Int),
		CodeHash: evm.EmptyCodeHash,
	}
}

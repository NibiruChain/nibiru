// Copyright (c) 2023-2024 Nibi, Inc.
package evmstate

import (
	"github.com/NibiruChain/collections"
	"github.com/holiman/uint256"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// ----------------------------------------------------------------------------
// StateDB Keeper implementation
// ----------------------------------------------------------------------------

// GetAccount: Ethereum account getter for an [evmstate.Account].
// Implements the `statedb.Keeper` interface.
// Returns nil if the account does not exist or has the wrong type.
func (k *Keeper) GetAccount(ctx sdk.Context, addr gethcommon.Address) *Account {
	acct := k.getAccountWithoutBalance(ctx, addr)
	if acct == nil {
		return nil
	}
	acct.BalanceNwei = k.GetWeiBalance(ctx, addr)
	return acct
}

// GetCode: Loads smart contract bytecode associated with the given code hash.
// Implements the `statedb.Keeper` interface.
func (k *Keeper) GetCode(ctx sdk.Context, codeHash gethcommon.Hash) []byte {
	if codeHash == evm.EmptyCodeHash {
		return nil
	}
	return k.EvmState.ContractBytecode.GetOr(ctx, codeHash.Bytes(), nil)
}

// ForEachStorage: Iterator over contract storage.
// Implements the `statedb.Keeper` interface.
func (k *Keeper) ForEachStorage(
	ctx sdk.Context,
	addr gethcommon.Address,
	stopIter func(key, value gethcommon.Hash) (keepGoing bool),
) {
	iter := k.EvmState.AccState.Iterate(
		ctx,
		collections.PairRange[gethcommon.Address, gethcommon.Hash]{}.Prefix(addr),
	)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		hash := iter.Key().K2()
		val := iter.Value()
		if !stopIter(hash, gethcommon.BytesToHash(val)) {
			return
		}
	}
}

// setAccBalance update account's balance, comparing with current balance first,
// then decides to ad or subtract based on what's needed.
// Implements the [Keeper] interface.
func (k *Keeper) setAccBalance(
	ctx sdk.Context, addr gethcommon.Address, newBal *uint256.Int,
) error {
	addrBech32 := eth.EthAddrToNibiruAddr(addr)
	balPre := k.GetWeiBalance(ctx, addr)

	cmpSign := newBal.Cmp(balPre)
	if cmpSign == 0 {
		return nil
	} else if cmpSign > 0 {
		balDelta := new(uint256.Int).Sub(newBal, balPre)
		k.Bank.AddWei(ctx, addrBech32, balDelta)
		return nil
	}
	balDelta := new(uint256.Int).Sub(balPre, newBal)
	if err := k.Bank.SubWei(ctx, addrBech32, balDelta); err != nil {
		return err
	}
	return nil
}

func (k *Keeper) SendWei(
	ctx sdk.Context, from gethcommon.Address, to gethcommon.Address, amtWei *uint256.Int,
) error {
	fromAddr := eth.EthAddrToNibiruAddr(from)
	toAddr := eth.EthAddrToNibiruAddr(to)
	if err := k.Bank.SubWei(ctx, fromAddr, amtWei); err != nil {
		return err
	}
	k.Bank.AddWei(ctx, toAddr, amtWei)
	return nil
}

// SetAccount: Updates nonce, balance, and codeHash.
// Implements the `statedb.Keeper` interface.
// Only called by `StateDB.Commit()`.
func (k *Keeper) SetAccount(
	ctx sdk.Context, addr gethcommon.Address, account Account,
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

	if err := k.setAccBalance(ctx, addr, account.BalanceNwei); err != nil {
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
// ------------------------------------------------------
// codeChange
// codeChange: [JournalChange] for an update to an account's code (smart contract
// bytecode). The previous code and hash for the code are stored to enable
// reversion.
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
	if err := k.setAccBalance(ctx, addr, new(uint256.Int)); err != nil {
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

// getAccountWithoutBalance load nonce and codehash without balance,
// more efficient in cases where balance is not needed.
func (k *Keeper) getAccountWithoutBalance(ctx sdk.Context, addr gethcommon.Address) *Account {
	acct := k.accountKeeper.GetAccount(ctx, eth.EthAddrToNibiruAddr(addr))
	if acct == nil {
		return nil
	}

	codeHash := evm.EmptyCodeHashBz
	ethAcct, ok := acct.(eth.EthAccountI)
	if ok {
		codeHash = ethAcct.GetCodeHash().Bytes()
	}

	return &Account{
		Nonce:    acct.GetSequence(),
		CodeHash: codeHash,
	}
}

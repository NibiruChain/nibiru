// Copyright (c) 2023-2024 Nibi, Inc.
package statedb

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// ExtStateDB defines an extension to the interface provided by the go-ethereum
// codebase to support additional state transition functionalities. In particular
// it supports appending a new entry to the state journal through
// AppendJournalEntry so that the state can be reverted after running
// stateful precompiled contracts.
type ExtStateDB interface {
	vm.StateDB
	AppendJournalEntry(JournalChange)
}

// Keeper provide underlying storage of StateDB
type Keeper interface {
	// GetAccount: Ethereum account getter for a [statedb.Account].
	GetAccount(ctx sdk.Context, addr common.Address) *Account
	GetState(ctx sdk.Context, addr common.Address, key common.Hash) common.Hash
	GetCode(ctx sdk.Context, codeHash common.Hash) []byte

	// ForEachStorage: Iterator over contract storage.
	ForEachStorage(
		ctx sdk.Context, addr common.Address,
		stopIter func(key, value common.Hash) bool,
	)

	SetAccount(ctx sdk.Context, addr common.Address, account Account) error
	SetState(ctx sdk.Context, addr common.Address, key common.Hash, value []byte)
	// SetCode: Setter for smart contract bytecode. Delete if code is empty.
	SetCode(ctx sdk.Context, codeHash []byte, code []byte)

	// DeleteAccount handles contract's suicide call, clearing the balance,
	// contract bytecode, contract state, and its native account.
	DeleteAccount(ctx sdk.Context, addr common.Address) error
}

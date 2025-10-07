package statedb

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

// Keeper provide underlying storage of StateDB
type Keeper interface {
	// GetAccount: Ethereum account getter for a [statedb.Account].
	GetAccount(ctx sdk.Context, addr gethcommon.Address) *Account
	GetState(ctx sdk.Context, addr gethcommon.Address, key gethcommon.Hash) gethcommon.Hash
	GetCode(ctx sdk.Context, codeHash gethcommon.Hash) []byte

	// GetAccNonce returns the sequence number of an account, returns 0 if the
	// account does not exist.
	GetAccNonce(ctx sdk.Context, addr gethcommon.Address) uint64

	BK() bankkeeper.Keeper
	// Bank() bankkeeper.BaseKeeper

	// ForEachStorage: Iterator over contract storage.
	ForEachStorage(
		ctx sdk.Context, addr gethcommon.Address,
		stopIter func(key, value gethcommon.Hash) bool,
	)

	SetAccount(ctx sdk.Context, addr gethcommon.Address, account Account) error
	SetState(ctx sdk.Context, addr gethcommon.Address, key gethcommon.Hash, value []byte)
	// SetCode: Setter for smart contract bytecode. Delete if code is empty.
	SetCode(ctx sdk.Context, codeHash []byte, code []byte)

	// DeleteAccount handles contract's suicide call, clearing the balance,
	// contract bytecode, contract state, and its native account.
	DeleteAccount(ctx sdk.Context, addr gethcommon.Address) error
}

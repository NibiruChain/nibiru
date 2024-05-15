// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

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

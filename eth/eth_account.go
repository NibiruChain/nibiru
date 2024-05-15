// Copyright (c) 2023-2024 Nibi, Inc.
package eth

import (
	"bytes"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	_ authtypes.AccountI                 = (*EthAccount)(nil)
	_ EthAccountI                        = (*EthAccount)(nil)
	_ authtypes.GenesisAccount           = (*EthAccount)(nil)
	_ codectypes.UnpackInterfacesMessage = (*EthAccount)(nil)
)

// EthAccType: Enum for Ethereum account types.
type EthAccType = int8

const (
	// EthAccType_EOA: For externally owned accounts (EOAs)
	EthAccType_EOA EthAccType = iota + 1
	// EthAccType_Contract: For smart contracts accounts.
	EthAccType_Contract
)

// EthAccountI represents the interface of an EVM compatible account
type EthAccountI interface { //revive:disable-line:exported
	authtypes.AccountI
	// EthAddress returns the ethereum Address representation of the AccAddress
	EthAddress() common.Address
	// CodeHash is the keccak256 hash of the contract code (if any)
	GetCodeHash() common.Hash
	// SetCodeHash sets the code hash to the account fields
	SetCodeHash(code common.Hash) error
	// Type returns the type of Ethereum Account (EOA or Contract)
	Type() EthAccType
}

func (acc EthAccount) GetBaseAccount() *authtypes.BaseAccount {
	return acc.BaseAccount
}

// EthAddress returns the account address ethereum format.
func (acc EthAccount) EthAddress() common.Address {
	return common.BytesToAddress(acc.GetAddress().Bytes())
}

func (acc EthAccount) GetCodeHash() common.Hash {
	return common.HexToHash(acc.CodeHash)
}

func (acc *EthAccount) SetCodeHash(codeHash common.Hash) error {
	acc.CodeHash = codeHash.Hex()
	return nil
}

// Type returns the type of Ethereum Account (EOA or Contract)
func (acc EthAccount) Type() EthAccType {
	if bytes.Equal(
		emptyCodeHash, common.HexToHash(acc.CodeHash).Bytes(),
	) {
		return EthAccType_EOA
	}
	return EthAccType_Contract
}

var emptyCodeHash = crypto.Keccak256(nil)

// ProtoBaseAccount: Implementation of `BaseAccount` for the `AccountI` interface
// used in the AccountKeeper from the Auth Module. [ProtoBaseAccount] is a
// drop-in replacement for the `auth.ProtoBaseAccount` from
// "cosmos-sdk/auth/types" extended to fit the the `EthAccountI` interface for
// Ethereum accounts.
func ProtoBaseAccount() authtypes.AccountI {
	return &EthAccount{
		BaseAccount: &authtypes.BaseAccount{},
		CodeHash:    common.BytesToHash(emptyCodeHash).String(),
	}
}

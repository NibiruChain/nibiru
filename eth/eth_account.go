// Copyright (c) 2023-2024 Nibi, Inc.
package eth

import (
	"bytes"
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func EthAddrToNibiruAddr(ethAddr gethcommon.Address) sdk.AccAddress {
	return ethAddr.Bytes()
}

func NibiruAddrToEthAddr(nibiruAddr sdk.AccAddress) gethcommon.Address {
	return gethcommon.BytesToAddress(nibiruAddr.Bytes())
}

// NibiruAddrFromStr parses the input string as an [sdk.AccAddress]. The argument
// can be an "0x" Ethereum hexadecimal or "nibi"-prefixed Bech32 address.
func NibiruAddrFromStr(addr string) (sdk.AccAddress, error) {
	errEthAddr := ValidateAddress(addr)
	addrBech32Maybe, errBech32 := sdk.AccAddressFromBech32(addr)
	switch {
	case errEthAddr == nil:
		return EthAddrToNibiruAddr(gethcommon.HexToAddress(addr)), nil
	case errBech32 == nil:
		return addrBech32Maybe, nil
	default:
		return sdk.AccAddress{}, fmt.Errorf(
			"could not parse address as Nibiru Bech32 or Ethereum hexadecimal: { given: %q, Ethereum error: %s, bech32 error: %s }",
			addr, errEthAddr, errBech32,
		)
	}
}

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
	EthAddress() gethcommon.Address
	// CodeHash is the keccak256 hash of the contract code (if any)
	GetCodeHash() gethcommon.Hash
	// SetCodeHash sets the code hash to the account fields
	SetCodeHash(code gethcommon.Hash) error
	// Type returns the type of Ethereum Account (EOA or Contract)
	Type() EthAccType
}

func (acc EthAccount) GetBaseAccount() *authtypes.BaseAccount {
	return acc.BaseAccount
}

// EthAddress returns the account address ethereum format.
func (acc EthAccount) EthAddress() gethcommon.Address {
	return gethcommon.BytesToAddress(acc.GetAddress().Bytes())
}

func (acc EthAccount) GetCodeHash() gethcommon.Hash {
	return gethcommon.HexToHash(acc.CodeHash)
}

func (acc *EthAccount) SetCodeHash(codeHash gethcommon.Hash) error {
	acc.CodeHash = codeHash.Hex()
	return nil
}

// Type returns the type of Ethereum Account (EOA or Contract)
func (acc EthAccount) Type() EthAccType {
	if bytes.Equal(
		emptyCodeHash, gethcommon.HexToHash(acc.CodeHash).Bytes(),
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
		CodeHash:    gethcommon.BytesToHash(emptyCodeHash).String(),
	}
}

// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"fmt"

	"github.com/cometbft/cometbft/crypto/tmhash"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
)

// FIXME: Explore problems arrising from ERC1155 creating multiple fungible
// tokens that are valid ERC20s with the same address.
// https://github.com/NibiruChain/nibiru/issues/1933
func (fun FunToken) ID() []byte {
	return NewFunTokenID(fun.Erc20Addr.Address, fun.BankDenom)
}

func NewFunTokenID(erc20 gethcommon.Address, bankDenom string) []byte {
	return tmhash.Sum([]byte(erc20.String() + "|" + bankDenom))
}

func funTokenValidationError(err error) error {
	return fmt.Errorf("FunTokenError: %s", err.Error())
}

func (fun FunToken) Validate() error {
	if err := sdk.ValidateDenom(fun.BankDenom); err != nil {
		return funTokenValidationError(err)
	}

	return nil
}

// NewFunToken is a canonical constructor for the [FunToken] type. Using this
// function helps guarantee a consistent string representation from the
// hex-encoded Ethereum address.
func NewFunToken(
	erc20 gethcommon.Address, bankDenom string, isMadeFromCoin bool,
) FunToken {
	return FunToken{
		Erc20Addr: eth.EIP55Addr{
			Address: erc20,
		},
		BankDenom:      bankDenom,
		IsMadeFromCoin: isMadeFromCoin,
	}
}

// ERC20Metadata: Optional metadata fields parsed from an ERC20 contract.
// The [Wrapped Ether contract] is a good example for reference.
//
//	```solidity
//	constract WETH9 {
//	  string public name     = "Wrapped Ether";
//	  string public symbol   = "WETH"
//	  uint8  public decimals = 18;
//	}
//	```
//
// Note that the name and symbol fields may be empty, according to the [ERC20
// specification].
//
// [Wrapped Ether contract]: https://etherscan.io/token/0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2#code
// [ERC20 specification]: https://eips.ethereum.org/EIPS/eip-20
type ERC20Metadata struct {
	Name     string
	Symbol   string
	Decimals uint8
}

// ToBankMetadata produces the "bank.Metadata" corresponding to a FunToken
// mapping created from an ERC20 token.
//
// The first argument of DenomUnits is required and the official base unit
// onchain, meaning the denom must be equivalent to bank.Metadata.Base.
//
// Decimals for an ERC20 are synonymous to "bank.DenomUnit.Exponent" in what
// they mean for external clients like wallets.
func (erc20Info ERC20Metadata) ToBankMetadata(
	bankDenom string, erc20 gethcommon.Address,
) bank.Metadata {
	var symbol string
	if erc20Info.Symbol != "" {
		symbol = erc20Info.Symbol
	} else {
		symbol = bankDenom
	}

	var name string
	if erc20Info.Name != "" {
		name = erc20Info.Name
	} else {
		name = bankDenom
	}

	denomUnits := []*bank.DenomUnit{
		{
			Denom:    bankDenom,
			Exponent: 0,
		},
	}
	display := symbol
	if erc20Info.Decimals > 0 {
		denomUnits = append(denomUnits, &bank.DenomUnit{
			Denom:    display,
			Exponent: uint32(erc20Info.Decimals),
		})
	}
	return bank.Metadata{
		Description: fmt.Sprintf(
			"ERC20 token \"%s\" represented as a Bank Coin with a corresponding FunToken mapping", erc20.String(),
		),
		DenomUnits: denomUnits,
		Base:       bankDenom,
		Display:    display,
		Name:       name,
		Symbol:     symbol,
	}
}

func ParseDecimalsFromBank(bankCoin bank.Metadata) uint8 {
	// bank.Metadata validation guarantees that both "Base" and "Display" denoms
	// pass "sdk.ValidateDenom" and that the "DenomUnits" slice has exponents in
	// ascending order with at least one element, which must be the base
	// denomination and have exponent 0.
	decimals := uint8(0)
	if len(bankCoin.DenomUnits) > 0 {
		decimalsIdx := len(bankCoin.DenomUnits) - 1
		decimals = uint8(bankCoin.DenomUnits[decimalsIdx].Exponent)
	}
	return decimals
}

// Checks that the necessary ERC20 metadata fields can be parsed from the given
// Bank Coin metadata for a FunToken mapping. ERC20.decimals can only be zero if
// "allowZeroDecimals" is true.
func ValidateFunTokenBankMetadata(
	bc bank.Metadata, allowZeroDecimals bool,
) (out ERC20Metadata, err error) {
	out = ERC20Metadata{
		Name:     bc.Name,
		Symbol:   bc.Symbol,
		Decimals: ParseDecimalsFromBank(bc),
	}
	if out.Name == "" {
		err = fmt.Errorf("empty name for ERC20")
		return
	} else if out.Symbol == "" {
		err = fmt.Errorf("empty symbol for ERC20")
		return
	} else if out.Decimals == 0 && !allowZeroDecimals {
		err = fmt.Errorf(`got ERC20.decimals = 0, which is consdiered an error unless "allow_zero_decimals" is true`)
		return
	}
	return out, nil
}

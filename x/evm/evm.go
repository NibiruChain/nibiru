// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"fmt"

	"github.com/cometbft/cometbft/crypto/tmhash"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
)

// FIXME: Explore problems arrising from ERC1155 creating multiple fungible
// tokens that are valid ERC20s with the same address.
// https://github.com/NibiruChain/nibiru/issues/1933
func (fun FunToken) ID() []byte {
	return NewFunTokenID(fun.Erc20Addr, fun.BankDenom)
}

func NewFunTokenID(erc20 eth.HexAddr, bankDenom string) []byte {
	return tmhash.Sum([]byte(erc20.String() + "|" + bankDenom))
}

func funTokenValidationError(err error) error {
	return fmt.Errorf("FunTokenError: %s", err.Error())
}

func (fun FunToken) Validate() error {
	if err := sdk.ValidateDenom(fun.BankDenom); err != nil {
		return funTokenValidationError(err)
	}

	if err := fun.Erc20Addr.Valid(); err != nil {
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
		Erc20Addr:      eth.NewHexAddr(erc20),
		BankDenom:      bankDenom,
		IsMadeFromCoin: isMadeFromCoin,
	}
}

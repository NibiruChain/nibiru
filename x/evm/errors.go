// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	codeErrInvalidState = uint32(iota) + 2 // NOTE: code 1 is reserved for internal errors
	codeErrZeroAddress
	codeErrInvalidAmount
	codeErrInvalidGasPrice
	codeErrInvalidGasFee
	codeErrInvalidRefund
	codeErrInvalidGasCap
	codeErrInvalidBaseFee
	codeErrInvalidAccount
	codeErrInactivePrecompile
)

var (
	// ErrInvalidState returns an error resulting from an invalid Storage State.
	ErrInvalidState = sdkioerrors.Register(ModuleName, codeErrInvalidState, "invalid storage state")

	// ErrZeroAddress returns an error resulting from an zero (empty) ethereum Address.
	ErrZeroAddress = sdkioerrors.Register(ModuleName, codeErrZeroAddress, "invalid zero address")

	// ErrInvalidAmount returns an error if a tx contains an invalid amount.
	ErrInvalidAmount = sdkioerrors.Register(ModuleName, codeErrInvalidAmount, "invalid transaction amount")

	// ErrInvalidGasPrice returns an error if an invalid gas price is provided to the tx.
	ErrInvalidGasPrice = sdkioerrors.Register(ModuleName, codeErrInvalidGasPrice, "invalid gas price")

	// ErrInvalidGasFee returns an error if the tx gas fee is out of bound.
	ErrInvalidGasFee = sdkioerrors.Register(ModuleName, codeErrInvalidGasFee, "invalid gas fee")

	// ErrInvalidRefund returns an error if the gas refund value is invalid.
	ErrInvalidRefund = sdkioerrors.Register(ModuleName, codeErrInvalidRefund, "invalid gas refund amount")

	// ErrInvalidGasCap returns an error if the gas cap value is negative or invalid
	ErrInvalidGasCap = sdkioerrors.Register(ModuleName, codeErrInvalidGasCap, "invalid gas cap")

	// ErrInvalidBaseFee returns an error if the base fee cap value is invalid
	ErrInvalidBaseFee = sdkioerrors.Register(ModuleName, codeErrInvalidBaseFee, "invalid base fee")

	// ErrInvalidAccount returns an error if the account is not an EVM compatible account
	ErrInvalidAccount = sdkioerrors.Register(ModuleName, codeErrInvalidAccount, "account type is not a valid ethereum account")
)

// NewRevertError unpacks the revert return bytes and returns a wrapped error
// with the return reason.
func NewRevertError(revertReason []byte) error {
	reason, unpackingError := abi.UnpackRevert(revertReason)

	if unpackingError != nil {
		return fmt.Errorf("execution reverted, but unable to parse reason \"%v\"", string(revertReason))
	}

	return fmt.Errorf("execution reverted with reason \"%v\"", reason)
}

// RevertError is an API error that encompass an EVM revert with JSON error
// code and a binary data blob.
type RevertError struct {
	error
}

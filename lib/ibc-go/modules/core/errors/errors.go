package errors

import (
	sdkioerrors "cosmossdk.io/errors"

	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
)

const codespace = exported.ModuleName

var (
	// ErrInvalidSequence is used the sequence number (nonce) is incorrect
	// for the signature.
	ErrInvalidSequence = sdkioerrors.Register(codespace, 1, "invalid sequence")

	// ErrUnauthorized is used whenever a request without sufficient
	// authorization is handled.
	ErrUnauthorized = sdkioerrors.Register(codespace, 2, "unauthorized")

	// ErrInsufficientFunds is used when the account cannot pay requested amount.
	ErrInsufficientFunds = sdkioerrors.Register(codespace, 3, "insufficient funds")

	// ErrUnknownRequest is used when the request body.
	ErrUnknownRequest = sdkioerrors.Register(codespace, 4, "unknown request")

	// ErrInvalidAddress is used when an address is found to be invalid.
	ErrInvalidAddress = sdkioerrors.Register(codespace, 5, "invalid address")

	// ErrInvalidCoins is used when sdk.Coins are invalid.
	ErrInvalidCoins = sdkioerrors.Register(codespace, 6, "invalid coins")

	// ErrOutOfGas is used when there is not enough gas.
	ErrOutOfGas = sdkioerrors.Register(codespace, 7, "out of gas")

	// ErrInvalidRequest defines an ABCI typed error where the request contains
	// invalid data.
	ErrInvalidRequest = sdkioerrors.Register(codespace, 8, "invalid request")

	// ErrInvalidHeight defines an error for an invalid height
	ErrInvalidHeight = sdkioerrors.Register(codespace, 9, "invalid height")

	// ErrInvalidVersion defines a general error for an invalid version
	ErrInvalidVersion = sdkioerrors.Register(codespace, 10, "invalid version")

	// ErrInvalidChainID defines an error when the chain-id is invalid.
	ErrInvalidChainID = sdkioerrors.Register(codespace, 11, "invalid chain-id")

	// ErrInvalidType defines an error an invalid type.
	ErrInvalidType = sdkioerrors.Register(codespace, 12, "invalid type")

	// ErrPackAny defines an error when packing a protobuf message to Any fails.
	ErrPackAny = sdkioerrors.Register(codespace, 13, "failed packing protobuf message to Any")

	// ErrUnpackAny defines an error when unpacking a protobuf message from Any fails.
	ErrUnpackAny = sdkioerrors.Register(codespace, 14, "failed unpacking protobuf message from Any")

	// ErrLogic defines an internal logic error, e.g. an invariant or assertion
	// that is violated. It is a programmer error, not a user-facing error.
	ErrLogic = sdkioerrors.Register(codespace, 15, "internal logic error")

	// ErrNotFound defines an error when requested entity doesn't exist in the state.
	ErrNotFound = sdkioerrors.Register(codespace, 16, "not found")
)

package types

import sdkioerrors "cosmossdk.io/errors"

var (
	ErrInvalid              = sdkioerrors.Register(ModuleName, 2, "invalid")
	ErrInvalidData          = sdkioerrors.Register(ModuleName, 3, "invalid data")
	ErrInvalidChecksum      = sdkioerrors.Register(ModuleName, 4, "invalid checksum")
	ErrInvalidClientMessage = sdkioerrors.Register(ModuleName, 5, "invalid client message")
	ErrRetrieveClientID     = sdkioerrors.Register(ModuleName, 6, "failed to retrieve client id")
	// Wasm specific
	ErrWasmEmptyCode                   = sdkioerrors.Register(ModuleName, 7, "empty wasm code")
	ErrWasmCodeTooLarge                = sdkioerrors.Register(ModuleName, 8, "wasm code too large")
	ErrWasmCodeExists                  = sdkioerrors.Register(ModuleName, 9, "wasm code already exists")
	ErrWasmChecksumNotFound            = sdkioerrors.Register(ModuleName, 10, "wasm checksum not found")
	ErrWasmSubMessagesNotAllowed       = sdkioerrors.Register(ModuleName, 11, "execution of sub messages is not allowed")
	ErrWasmEventsNotAllowed            = sdkioerrors.Register(ModuleName, 12, "returning events from a contract is not allowed")
	ErrWasmAttributesNotAllowed        = sdkioerrors.Register(ModuleName, 13, "returning attributes from a contract is not allowed")
	ErrWasmContractCallFailed          = sdkioerrors.Register(ModuleName, 14, "wasm contract call failed")
	ErrWasmInvalidResponseData         = sdkioerrors.Register(ModuleName, 15, "wasm contract returned invalid response data")
	ErrWasmInvalidContractModification = sdkioerrors.Register(ModuleName, 16, "wasm contract made invalid state modifications")
)

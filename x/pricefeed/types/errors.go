package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/pricefeed module sentinel errors
var (
	ErrSample = sdkerrors.Register(ModuleName, 1100, "Sample error")
	// ErrEmptyInput error for empty input
	ErrEmptyInput = sdkerrors.Register(ModuleName, 2, "Input must not be empty")
	// ErrExpired error for posted price messages with expired price
	ErrExpired = sdkerrors.Register(ModuleName, 3, "Price is expired")
	// ErrNoValidPrice error for posted price messages with expired price
	ErrNoValidPrice = sdkerrors.Register(ModuleName, 4, "All input prices are expired")
	// ErrInvalidMarket error for posted price messages for invalid markets
	ErrInvalidMarket = sdkerrors.Register(ModuleName, 5, "Market does not exist")
	// ErrInvalidOracle error for posted price messages for invalid oracles
	ErrInvalidOracle = sdkerrors.Register(ModuleName, 6, "Oracle does not exist or not authorized")
	// ErrAssetNotFound error for not found asset
	ErrAssetNotFound = sdkerrors.Register(ModuleName, 7, "Asset not found")
)

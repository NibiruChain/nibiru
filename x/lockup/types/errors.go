package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/lockup module sentinel errors.
var (
	ErrNotLockOwner   = sdkerrors.Register(ModuleName, 1, "msg sender is not the owner of specified lock")
	ErrLockupNotFound = sdkerrors.Register(ModuleName, 2, "lockup not found")
	ErrLockEndTime    = sdkerrors.Register(ModuleName, 3, "lock end time not met")
)

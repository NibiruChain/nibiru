package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrMinLockupDurationTooLow        = sdkerrors.Register(ModuleName, 1, "minimum lockup duration too low")
	ErrStartTimeInPast                = sdkerrors.Register(ModuleName, 2, "incentivization program start time is in the past")
	ErrEpochsTooLow                   = sdkerrors.Register(ModuleName, 3, "number of epochs too low")
	ErrIncentivizationProgramNotFound = sdkerrors.Register(ModuleName, 4, "incentivization program not found")
)

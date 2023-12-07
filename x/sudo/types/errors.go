package types

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrUnauthorized = sdkerrors.Register(ModuleName, 2, "unauthorized: missing sudo permissions")
	errGenesis      = sdkerrors.Register(ModuleName, 3, "sudo genesis error")
	errSudoers      = sdkerrors.Register(ModuleName, 4, "sudoers error")
)

func ErrGenesis(errMsg string) error {
	return fmt.Errorf("%s: %s", errGenesis, errMsg)
}
func ErrSudoers(errMsg string) error {
	return fmt.Errorf("%s: %s", errSudoers, errMsg)
}

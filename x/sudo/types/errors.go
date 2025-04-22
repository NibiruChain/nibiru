package types

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
)

var (
	ErrUnauthorized = sdkioerrors.Register(ModuleName, 2, "unauthorized: missing sudo permissions")
	errGenesis      = sdkioerrors.Register(ModuleName, 3, "sudo genesis error")
	errSudoers      = sdkioerrors.Register(ModuleName, 4, "sudoers error")
)

func ErrGenesis(errMsg string) error {
	return fmt.Errorf("%s: %s", errGenesis, errMsg)
}

func ErrSudoers(errMsg string) error {
	return fmt.Errorf("%s: %s", errSudoers, errMsg)
}

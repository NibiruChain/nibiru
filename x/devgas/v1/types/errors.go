package types

import (
	sdkioerrors "cosmossdk.io/errors"
)

// errors
var (
	ErrFeeShareDisabled              = sdkioerrors.Register(ModuleName, 1, "feeshare module is disabled by governance")
	ErrFeeShareAlreadyRegistered     = sdkioerrors.Register(ModuleName, 2, "feeshare already exists for given contract")
	ErrFeeShareNoContractDeployed    = sdkioerrors.Register(ModuleName, 3, "no contract deployed")
	ErrFeeShareContractNotRegistered = sdkioerrors.Register(ModuleName, 4, "no feeshare registered for contract")
	ErrFeeSharePayment               = sdkioerrors.Register(ModuleName, 5, "feeshare payment error")
	ErrFeeShareInvalidWithdrawer     = sdkioerrors.Register(ModuleName, 6, "invalid withdrawer address")
)

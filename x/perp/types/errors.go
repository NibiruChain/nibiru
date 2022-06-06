package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

// DONTCOVER

// x/perp module sentinel errors
var (
	ErrSample = sdkerrors.Register(ModuleName, 1, "sample error")
)

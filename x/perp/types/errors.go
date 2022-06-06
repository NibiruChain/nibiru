package types

import (
	"errors"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DONTCOVER

// x/perp module sentinel errors
var (
	ErrMarginHighEnough = sdkerrors.Register(ModuleName, 1,
		"Margin is higher than required maintenant margin ratio")
	ErrPositionNotFound     = errors.New("no position found")
	ErrPairNotFound         = errors.New("pair doesn't have live vpool")
	ErrPairMetadataNotFound = errors.New("pair doesn't have metadata")
	ErrPositionZero         = errors.New("position is zero")
)

package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrPairNotSupported     = sdkerrors.Register(ModuleName, 1, "pair not supported")
	ErrOverTradingLimit     = sdkerrors.Register(ModuleName, 2, "over trading limit")
	ErrQuoteReserveAtZero   = sdkerrors.Register(ModuleName, 3, "quote reserve after at zero")
	ErrBaseReserveAtZero    = sdkerrors.Register(ModuleName, 4, "base reserve after at zero")
	ErrNoLastSnapshotSaved  = sdkerrors.Register(ModuleName, 5, "There was no last snapshot, could be that you did not do snapshot on pool creation")
	ErrOverFluctuationLimit = sdkerrors.Register(ModuleName, 6, "price is over fluctuation limit")
	ErrAssetOverUserLimit   = sdkerrors.Register(ModuleName, 7, "amount of assets traded is over user-defined limit")
	ErrOpeningPosition      = sdkerrors.Register(ModuleName, 8, "error opening position")
	ErrClosingPosition      = sdkerrors.Register(ModuleName, 9, "error closing position")
	ErrNoValidPrice         = sdkerrors.Register(ModuleName, 10, "no valid prices available")
	ErrNoValidTWAP          = sdkerrors.Register(ModuleName, 11, "TWAP price not found")
)

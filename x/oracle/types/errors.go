package types

import (
	"fmt"

	"github.com/cometbft/cometbft/crypto/tmhash"

	"sync/atomic"

	sdkerrors "cosmossdk.io/errors"
)

var moduleErrorCodeIdx uint32 = 1

// registerError: Cleaner way of using 'sdkerrors.Register' without as much time
// manually writing integers.
func registerError(msg string) *sdkerrors.Error {
	// Atomic for thread safety on concurrent calls
	atomic.AddUint32(&moduleErrorCodeIdx, 1)
	return sdkerrors.Register(ModuleName, moduleErrorCodeIdx, msg)
}

// Oracle Errors
var (
	ErrInvalidExchangeRate = registerError("invalid exchange rate")
	ErrNoPrevote           = registerError("no prevote")
	ErrNoVote              = registerError("no vote")
	ErrNoVotingPermission  = registerError("unauthorized voter")
	ErrInvalidHash         = registerError("invalid hash")
	ErrInvalidHashLength   = registerError(
		fmt.Sprintf("invalid hash length; should equal %d", tmhash.TruncatedSize))
	ErrHashVerificationFailed = registerError("hash verification failed")
	ErrRevealPeriodMissMatch  = registerError("reveal period of submitted vote do not match with registered prevote")
	ErrInvalidSaltLength      = registerError("invalid salt length; should be 1~4")
	ErrNoAggregatePrevote     = registerError("no aggregate prevote")
	ErrNoAggregateVote        = registerError("no aggregate vote")
	ErrUnknownPair            = registerError("unknown pair")
	ErrNoValidTWAP            = registerError("TWA price not found")
)

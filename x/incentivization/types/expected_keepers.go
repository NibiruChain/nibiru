package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ltypes "github.com/NibiruChain/nibiru/x/lockup/types"
)

type LockupKeeper interface {
	LocksByDenomUnlockingAfter(ctx sdk.Context, denom string, duration time.Duration, do func(lock *ltypes.Lock) (stop bool))
}

package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewLock returns a new instance of period lock.
func NewLock(lockId uint64, owner sdk.AccAddress, duration time.Duration,
	endTime time.Time, coins sdk.Coins) Lock {
	return Lock{
		LockId:   lockId,
		Owner:    owner.String(),
		Duration: duration,
		EndTime:  endTime,
		Coins:    coins,
	}
}

// OwnerAddress returns locks owner address.
func (p Lock) OwnerAddress() sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(p.Owner)
	if err != nil {
		panic(err)
	}
	return addr
}

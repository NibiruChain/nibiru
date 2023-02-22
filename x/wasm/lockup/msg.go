package lockup

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Lockup ExecMsgs
type LockupMsg struct {
	Lock           *Lock           `json:"lock,omitempty"`
	InitiateUnlock *InitiateUnlock `json:"initiate_unlock,omitempty"`
	WithdrawFunds  *WithdrawFunds  `json:"withdraw_funds,omitempty"`
}

type Lock struct {
	Blocks int `json:"blocks"`
}

type InitiateUnlock struct {
	Id int `json:"id"`
}

type WithdrawFunds struct {
	Id int `json:"id"`
}

// Lockup QueryMsgs
type LockupQuery struct {
	LocksByDenomUnlockingAfter           *LocksByDenomUnlockingAfter           `json:"locks_by_denom_unlocking_after,omitempty"`
	LocksByDenomAndAddressUnlockingAfter *LocksByDenomAndAddressUnlockingAfter `json:"locks_by_denom_and_address_unlocking_after,omitempty"`
	LocksByDenomBetween                  *LocksByDenomBetween                  `json:"locks_by_denom_between,omitempty"`
	LocksByDenomAndAddressBetween        *LocksByDenomAndAddressBetween        `json:"locks_by_denom_and_address_between,omitempty"`
}

type LocksByDenomUnlockingAfter struct {
	Denom          string `json:"denom"`
	UnlockingAfter int    `json:"unlocking_after"`
}

type LocksByDenomAndAddressUnlockingAfter struct {
	Denom          string `json:"denom"`
	UnlockingAfter int    `json:"unlocking_after"`
	Address        string `json:"address"`
}

type LocksByDenomBetween struct {
	Denom          string `json:"denom"`
	LockedBefore   int    `json:"locked_before"`
	UnlockingAfter int    `json:"unlocking_after"`
}

type LocksByDenomAndAddressBetween struct {
	Denom          string `json:"denom"`
	Address        string `json:"address"`
	LockedBefore   int    `json:"locked_before"`
	UnlockingAfter int    `json:"unlocking_after"`
}

// Responses
type LocksByDenomUnlockingAfterResponse []LockResponse

type LockResponse struct {
	Id             int      `json:"id"`
	Coin           sdk.Coin `json:"coin"`
	Owner          string   `json:"owner"`
	DurationBlocks int      `json:"duration_blocks"`
	StartBlock     int      `json:"start_block"`
	EndBlock       uint64   `json:"end_block"`
	FundsWithdrawn bool     `json:"funds_withdrawn"`
}

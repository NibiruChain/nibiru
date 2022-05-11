package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// constants.
const (
	TypeMsgLockTokens        = "lock_tokens"
	TypeMsgBeginUnlockingAll = "begin_unlocking_all"
	TypeMsgBeginUnlocking    = "begin_unlocking"
)

var (
	_ sdk.Msg = (*MsgLockTokens)(nil)
	_ sdk.Msg = (*MsgInitiateUnlock)(nil)
)

// NewMsgLockTokens creates a message to lock tokens.
func NewMsgLockTokens(owner sdk.AccAddress, duration time.Duration, coins sdk.Coins) *MsgLockTokens {
	return &MsgLockTokens{
		Owner:    owner.String(),
		Duration: duration,
		Coins:    coins,
	}
}

func (m MsgLockTokens) Route() string { return RouterKey }
func (m MsgLockTokens) Type() string  { return TypeMsgLockTokens }
func (m MsgLockTokens) ValidateBasic() error {
	if m.Duration <= 0 {
		return fmt.Errorf("duration should be positive: %d < 0", m.Duration)
	}
	return nil
}

func (m MsgLockTokens) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgLockTokens) GetSigners() []sdk.AccAddress {
	owner, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{owner}
}

func (m *MsgInitiateUnlock) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Owner)
	if err != nil {
		return err
	}
	return nil
}

func (m *MsgInitiateUnlock) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Owner)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{addr}
}

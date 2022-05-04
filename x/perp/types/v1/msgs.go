package v1

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgFoo{}

const (
	TypeMsgFoo = "foo_msg"
)

func (m MsgFoo) Route() string { return RouterKey }
func (m MsgFoo) Type() string  { return TypeMsgFoo }

func (m MsgFoo) ValidateBasic() error {
	return nil
}

func (m MsgFoo) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgFoo) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

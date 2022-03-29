package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgBurn = "mint"

var _ sdk.Msg = &MsgBurnStable{}

func NewMsgBurn(creator string, coin sdk.Coin) *MsgBurnStable {
	return &MsgBurnStable{
		Creator: creator,
		Stable:  coin,
	}
}

func (msg *MsgBurnStable) Route() string {
	return RouterKey
}

func (msg *MsgBurnStable) Type() string {
	return TypeMsgBurn
}

func (msg *MsgBurnStable) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgBurnStable) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgBurnStable) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

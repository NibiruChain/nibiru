package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgMintStable = "mint"

var _ sdk.Msg = &MsgMintStable{}

func NewMsgMintStable(creator string, coin sdk.Coin) *MsgMintStable {
	return &MsgMintStable{
		Creator: creator,
		Stable:  coin,
	}
}

func (msg *MsgMintStable) Route() string {
	return RouterKey
}

func (msg *MsgMintStable) Type() string {
	return TypeMsgMintStable
}

func (msg *MsgMintStable) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgMintStable) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgMintStable) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

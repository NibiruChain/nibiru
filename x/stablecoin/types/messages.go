package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

// ----------------------------------------------------------------
// MsgMintStable
// ----------------------------------------------------------------

var (
	_ sdk.Msg = &MsgMintStable{}
	_ sdk.Msg = &MsgBurnStable{}
	_ sdk.Msg = &MsgRecollateralize{}
	_ sdk.Msg = &MsgBuyback{}
)

func (msg *MsgMintStable) Route() string {
	return RouterKey
}

func (msg *MsgMintStable) Type() string {
	return "mint-stable"
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
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

// ----------------------------------------------------------------
// MsgBurnStable
// ----------------------------------------------------------------

var _ sdk.Msg = &MsgBurnStable{}

func NewMsgBurn(creator string, coin sdk.Coin) *MsgBurnStable {
	return &MsgBurnStable{
		Creator: creator,
		Stable:  coin,
	}
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
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

// ----------------------------------------------------------------
// MsgRecollateralize
// ----------------------------------------------------------------

var _ sdk.Msg = &MsgRecollateralize{}

func NewMsgRecollateralize(creator string, coin sdk.Coin) *MsgRecollateralize {
	return &MsgRecollateralize{
		Creator: creator,
		Coll:    coin,
	}
}

func (msg *MsgRecollateralize) Route() string {
	return RouterKey
}

func (msg *MsgRecollateralize) Type() string {
	return "recoll"
}

func (msg *MsgRecollateralize) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgRecollateralize) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRecollateralize) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

// ----------------------------------------------------------------
// MsgBuyback
// ----------------------------------------------------------------

var _ sdk.Msg = &MsgBuyback{}

func NewMsgBuyback(creator string, coin sdk.Coin) *MsgBuyback {
	return &MsgBuyback{
		Creator: creator,
		Gov:     coin,
	}
}

func (msg *MsgBuyback) Route() string {
	return RouterKey
}

func (msg *MsgBuyback) Type() string {
	return "recoll"
}

func (msg *MsgBuyback) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgBuyback) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgBuyback) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

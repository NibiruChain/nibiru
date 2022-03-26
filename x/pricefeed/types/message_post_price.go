package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	// TypeMsgPostPrice type of PostPrice msg
	TypeMsgPostPrice = "post_price"

	// MaxExpiry defines the max expiry time defined as UNIX time (9999-12-31 23:59:59 +0000 UTC)
	MaxExpiry = 253402300799
)

var _ sdk.Msg = &MsgPostPrice{}

func NewMsgPostPrice(creator string, marketId string, price sdk.Dec, expiry time.Time) *MsgPostPrice {
	return &MsgPostPrice{
		From:     creator,
		MarketID: marketId,
		Price:    price,
		Expiry:   expiry,
	}
}

func (msg *MsgPostPrice) Route() string {
	return RouterKey
}

func (msg *MsgPostPrice) Type() string {
	return TypeMsgPostPrice
}

func (msg *MsgPostPrice) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPostPrice) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPostPrice) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgJoinPool = "create_pool"

var _ sdk.Msg = &MsgJoinPool{}

func NewMsgJoinPool(sender string, poolId uint64, tokensIn sdk.Coins) *MsgJoinPool {
	return &MsgJoinPool{
		Sender:   sender,
		PoolId:   poolId,
		TokensIn: tokensIn,
	}
}

func (msg *MsgJoinPool) Route() string {
	return RouterKey
}

func (msg *MsgJoinPool) Type() string {
	return TypeMsgJoinPool
}

func (msg *MsgJoinPool) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgJoinPool) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgJoinPool) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

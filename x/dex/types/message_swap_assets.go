package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSwapAssets = "swap_assets"

var _ sdk.Msg = &MsgSwapAssets{}

func NewMsgSwapAssets(sender string, poolId uint64, tokenIn sdk.Coin, tokenOutDenom string) *MsgSwapAssets {
	return &MsgSwapAssets{
		Sender:        sender,
		PoolId:        poolId,
		TokenIn:       tokenIn,
		TokenOutDenom: tokenOutDenom,
	}
}

func (msg *MsgSwapAssets) Route() string {
	return RouterKey
}

func (msg *MsgSwapAssets) Type() string {
	return TypeMsgSwapAssets
}

func (msg *MsgSwapAssets) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSwapAssets) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSwapAssets) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.PoolId == 0 {
		return ErrInvalidPoolId.Wrapf("pool id cannot be %d", msg.PoolId)
	}

	if msg.TokenIn.Amount.LTE(sdk.ZeroInt()) {
		return ErrInvalidTokenIn.Wrapf("invalid argument %s", msg.TokenIn.String())
	}

	if msg.TokenOutDenom == "" {
		return ErrInvalidTokenOutDenom.Wrap("cannot be empty")
	}

	return nil
}

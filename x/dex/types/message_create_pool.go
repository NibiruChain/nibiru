package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgCreatePool = "create_pool"

var _ sdk.Msg = &MsgCreatePool{}

func NewMsgCreatePool(creator string, poolAssets []PoolAsset, poolParams *PoolParams) *MsgCreatePool {
	return &MsgCreatePool{
		Creator:    creator,
		PoolAssets: poolAssets,
		PoolParams: poolParams,
	}
}

func (msg *MsgCreatePool) Route() string {
	return RouterKey
}

func (msg *MsgCreatePool) Type() string {
	return TypeMsgCreatePool
}

func (msg *MsgCreatePool) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCreatePool) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreatePool) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.PoolAssets) < MinPoolAssets {
		return ErrTooFewPoolAssets.Wrapf("invalid number of assets (%d)", len(msg.PoolAssets))
	}
	if len(msg.PoolAssets) > MaxPoolAssets {
		return ErrTooManyPoolAssets.Wrapf("invalid number of assets (%d)", len(msg.PoolAssets))
	}

	for _, asset := range msg.PoolAssets {
		if asset.Weight.LTE(sdk.ZeroInt()) {
			return ErrInvalidTokenWeight.Wrapf("invalid token weight %d for denom %s", asset.Weight, asset.Token.Denom)
		}
	}

	if msg.PoolParams.SwapFee.LT(sdk.ZeroDec()) || msg.PoolParams.SwapFee.GT(sdk.OneDec()) {
		return ErrInvalidSwapFee.Wrapf("invalid swap fee: %s", msg.PoolParams.SwapFee)
	}

	if msg.PoolParams.ExitFee.LT(sdk.ZeroDec()) || msg.PoolParams.ExitFee.GT(sdk.OneDec()) {
		return ErrInvalidExitFee.Wrapf("invalid exit fee: %s", msg.PoolParams.ExitFee)
	}

	return nil
}

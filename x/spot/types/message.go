package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgExitPool   = "exit_pool"
	TypeMsgJoinPool   = "join_pool"
	TypeMsgSwapAssets = "swap_assets"
	TypeMsgCreatePool = "create_pool"
)

var (
	_ sdk.Msg = &MsgExitPool{}
	_ sdk.Msg = &MsgJoinPool{}
	_ sdk.Msg = &MsgSwapAssets{}
	_ sdk.Msg = &MsgCreatePool{}
)

func NewMsgExitPool(sender string, poolId uint64, poolShares sdk.Coin) *MsgExitPool {
	return &MsgExitPool{
		Sender:     sender,
		PoolId:     poolId,
		PoolShares: poolShares,
	}
}

func (msg *MsgExitPool) Route() string {
	return RouterKey
}

func (msg *MsgExitPool) Type() string {
	return TypeMsgExitPool
}

func (msg *MsgExitPool) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgExitPool) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgExitPool) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid address (%s)", err)
	}
	return nil
}

var _ sdk.Msg = &MsgJoinPool{}

func NewMsgJoinPool(sender string, poolId uint64, tokensIn sdk.Coins, useAllCoins bool) *MsgJoinPool {
	return &MsgJoinPool{
		Sender:      sender,
		PoolId:      poolId,
		TokensIn:    tokensIn,
		UseAllCoins: useAllCoins,
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
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

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
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid creator address (%s)", err)
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
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid creator address (%s)", err)
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

	if (msg.PoolParams.PoolType != PoolType_STABLESWAP) && (msg.PoolParams.PoolType != PoolType_BALANCER) {
		return ErrInvalidPoolType
	}

	if msg.PoolParams.PoolType == PoolType_STABLESWAP {
		if msg.PoolParams.A.IsNil() {
			return ErrAmplificationMissing
		}

		if !msg.PoolParams.A.IsPositive() {
			return ErrAmplificationTooLow
		}
	}

	return nil
}

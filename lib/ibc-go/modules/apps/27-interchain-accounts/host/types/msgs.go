package types

import (
	sdkioerrors "cosmossdk.io/errors"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	ibcerrors "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/errors"
)

var _ sdk.Msg = (*MsgModuleQuerySafe)(nil)

// NewMsgModuleQuerySafe creates a new MsgModuleQuerySafe instance
func NewMsgModuleQuerySafe(signer string, requests []*QueryRequest) *MsgModuleQuerySafe {
	return &MsgModuleQuerySafe{
		Signer:   signer,
		Requests: requests,
	}
}

// ValidateBasic implements sdk.HasValidateBasic
func (msg MsgModuleQuerySafe) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkioerrors.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	if len(msg.Requests) == 0 {
		return sdkioerrors.Wrapf(ibcerrors.ErrInvalidRequest, "no queries provided")
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgModuleQuerySafe) GetSigners() []sdk.AccAddress {
	accAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{accAddr}
}

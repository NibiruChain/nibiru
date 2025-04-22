package types

import (
	sdkioerrors "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic runs stateless checks on the query requests
func (q QueryFeeShareRequest) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(q.ContractAddress); err != nil {
		return sdkioerrors.Wrapf(err, "invalid contract address %s", q.ContractAddress)
	}
	return nil
}

// ValidateBasic runs stateless checks on the query requests
func (q QueryFeeSharesRequest) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(q.Deployer); err != nil {
		return sdkioerrors.Wrapf(err, "invalid deployer address %s", q.Deployer)
	}
	return nil
}

// ValidateBasic runs stateless checks on the query requests
func (q QueryFeeSharesByWithdrawerRequest) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(q.WithdrawerAddress); err != nil {
		return sdkioerrors.Wrapf(err, "invalid withdraw address %s", q.WithdrawerAddress)
	}
	return nil
}

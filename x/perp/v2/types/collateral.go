package types

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Errors
var ErrInvalidCollateral = sdkerrors.Register("collateral", 1, "invalid token factory")

// Default collateral used for testing only.
var DefaultTestingCollateralNotForProd = NewCollateral("cosmos15u3dt79t6sxxa3x3kpkhzsy56edaa5a66wvt3kxmukqjz2sx0hesh45zsv", "unusd")

func NewCollateral(contractAddress string, collateral string) Collateral {
	return Collateral{
		ContractAddress: contractAddress,
		Denom:           collateral,
	}
}

func TryNewCollateral(s string) (Collateral, error) {
	split := strings.Split(s, "/")
	if len(split) != 3 {
		return Collateral{}, ErrInvalidCollateral
	}
	collateral := NewCollateral(split[1], split[2])
	return collateral, collateral.Validate()
}

func MustNewCollateral(s string) Collateral {
	collateral, err := TryNewCollateral(s)
	if err != nil {
		panic(err)
	}
	return collateral
}

func (c Collateral) Validate() error {
	if _, err := sdk.AccAddressFromBech32(c.ContractAddress); err != nil {
		return err
	}
	return nil
}

func (c Collateral) UpdatedContractAddress(contractAddress string) (Collateral, error) {
	newCollateral := Collateral{
		ContractAddress: contractAddress,
		Denom:           c.Denom,
	}

	return newCollateral, newCollateral.Validate()
}

func (collateral Collateral) Equal(other Collateral) bool {
	return collateral.Denom == other.Denom && collateral.ContractAddress == other.ContractAddress
}

// GetTFDenom returns the token factory denom for the collateral
// The format is tf/{contractAddress}/{denom}
func (collateral Collateral) GetTFDenom() string {
	return "tf/" + collateral.ContractAddress + "/" + collateral.Denom
}

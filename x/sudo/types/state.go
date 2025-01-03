package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (sudo Sudoers) Validate() error {
	if _, err := sdk.AccAddressFromBech32(sudo.Root); err != nil {
		return ErrSudoers("root addr: " + err.Error())
	}
	for _, contract := range sudo.Contracts {
		if _, err := sdk.AccAddressFromBech32(contract); err != nil {
			return ErrSudoers("contract addr: " + err.Error())
		}
	}
	return nil
}

type SudoersJson struct {
	Root      string   `json:"root"`
	Contracts []string `json:"contracts"`
}

package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (sudo Sudoers) Validate() error {
	for _, contract := range sudo.Contracts {
		if _, err := sdk.AccAddressFromBech32(contract); err != nil {
			return err
		}
	}
	return nil
}

type SudoersJson struct {
	Root      string   `json:"root"`
	Contracts []string `json:"contracts"`
}

func (sudo Sudoers) String() string {
	jsonBz, _ := json.Marshal(SudoersJson(sudo))
	return string(jsonBz)
}

package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalTypeWhitelistPriceOracle string = "WhitelistPriceOracle"
)

func NewWhitelistPriceOracleProposal(title string, description string, oracleAddress sdk.AccAddress) *WhitelistPriceOracleProposal {
	return &WhitelistPriceOracleProposal{Title: title, Description: description, OracleAddress: oracleAddress.String()}
}

var _ govtypes.Content = &WhitelistPriceOracleProposal{}

func (m *WhitelistPriceOracleProposal) ProposalRoute() string { return RouterKey }

func (m *WhitelistPriceOracleProposal) ProposalType() string { return ProposalTypeWhitelistPriceOracle }

func (m *WhitelistPriceOracleProposal) ValidateBasic() error {
	return govtypes.ValidateAbstract(m)
}

func (m *WhitelistPriceOracleProposal) String() string {
	return fmt.Sprintf(`Whitelist Price Oracle:
  Title:       %s
  Description: %s
`, m.Title, m.Description)
}

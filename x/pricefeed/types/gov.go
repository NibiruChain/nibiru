package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/common"
)

const (
	ProposalTypeAddOracle = "AddOracle"
)

var _ govtypes.Content = &AddOracleProposal{}

func init() {
	govtypes.RegisterProposalType(ProposalTypeAddOracle)
	govtypes.RegisterProposalTypeCodec(&AddOracleProposal{}, "nibiru/AddOracleProposal")
}

func NewAddOracleProposal(
	title string, description string, oracles []string, pairs []string,
) *AddOracleProposal {
	proposal := &AddOracleProposal{
		Title:       title,
		Description: description,
		Oracles:     oracles,
		Pairs:       pairs,
	}

	if err := proposal.Validate(); err != nil {
		panic(err)
	}

	return proposal
}

func (m *AddOracleProposal) ProposalRoute() string {
	return RouterKey
}

func (m *AddOracleProposal) ProposalType() string {
	return ProposalTypeAddOracle
}

func (m *AddOracleProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(m); err != nil {
		return err
	}

	if len(m.Pairs) == 0 {
		return fmt.Errorf("can't whitelist an oracle address without pairs")
	}

	return nil
}

func (m *AddOracleProposal) Validate() error {
	seenOracles := make(map[string]bool)
	for _, oracleStr := range m.Oracles {
		_, err := sdk.AccAddressFromBech32(oracleStr)
		if err != nil {
			return err
		}

		if seenOracles[oracleStr] {
			continue
		}
		seenOracles[oracleStr] = true
	}

	for _, pairStr := range m.Pairs {
		_, err := common.NewAssetPair(pairStr)
		if err != nil {
			return err
		}
	}

	return nil
}

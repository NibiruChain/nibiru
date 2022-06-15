package types

import (
	"fmt"

	"github.com/NibiruChain/nibiru/x/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
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
	title string, description string, oracle string, pairs []string,
) *AddOracleProposal {

	proposal := &AddOracleProposal{
		Title:       title,
		Description: description,
		Oracle:      oracle,
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
	_, err := sdk.AccAddressFromBech32(m.Oracle)
	if err != nil {
		return err
	}

	for _, pairStr := range m.Pairs {
		_, err = common.NewAssetPairFromStr(pairStr)
		if err != nil {
			return err
		}
	}

	return nil
}

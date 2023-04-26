package v2

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalTypeCreatePool         = "CreatePoolV2"
	ProposalTypeEditPoolConfig     = "EditPoolConfigV2"
	ProposalTypeEditSwapInvariants = "EditSwapInvariantsV2"
)

var _ govtypes.Content = &CreatePoolProposal{}
var _ govtypes.Content = &EditPoolConfigProposal{}
var _ govtypes.Content = &EditSwapInvariantsProposal{}

func init() {
	govtypes.RegisterProposalType(ProposalTypeCreatePool)
	govtypes.RegisterProposalTypeCodec(&CreatePoolProposal{}, "nibiru/CreatePoolProposalV2")
	govtypes.RegisterProposalType(ProposalTypeEditPoolConfig)
	govtypes.RegisterProposalTypeCodec(&EditPoolConfigProposal{}, "nibiru/EditPoolConfigProposalV2")
	govtypes.RegisterProposalType(ProposalTypeEditSwapInvariants)
	govtypes.RegisterProposalTypeCodec(&EditSwapInvariantsProposal{}, "nibiru/EditSwapInvariantsProposalV2")
}

// CreatePoolProposal

func (proposal *CreatePoolProposal) ProposalRoute() string {
	return "perp"
}

func (proposal *CreatePoolProposal) ProposalType() string {
	return ProposalTypeCreatePool
}

func (proposal *CreatePoolProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(proposal); err != nil {
		return err
	}

	err := proposal.Pair.Validate()
	if err != nil {
		return err
	}

	err = proposal.Market.Validate()
	if err != nil {
		return err
	}

	err = proposal.Amm.Validate()
	if err != nil {
		return err
	}

	return nil
}

// EditPoolConfigProposal

func (proposal *EditPoolConfigProposal) ProposalRoute() string {
	return "perp"
}

func (proposal *EditPoolConfigProposal) ProposalType() string {
	return ProposalTypeEditPoolConfig
}

func (proposal *EditPoolConfigProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(proposal); err != nil {
		return err
	}

	err := proposal.Pair.Validate()
	if err != nil {
		return err
	}

	err = proposal.Market.Validate()
	if err != nil {
		return err
	}

	err = proposal.Amm.Validate()
	if err != nil {
		return err
	}

	return nil
}

// EditSwapInvariantsProposal

func (proposal *EditSwapInvariantsProposal) ProposalRoute() string {
	return "perp"
}

func (proposal *EditSwapInvariantsProposal) ProposalType() string {
	return ProposalTypeEditSwapInvariants
}

func (proposal *EditSwapInvariantsProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(proposal); err != nil {
		return err
	}

	for _, kv := range proposal.SwapInvariantMaps {
		if err := kv.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (kv *EditSwapInvariantsProposal_SwapInvariantMultiple) Validate() error {
	var comboError []string
	err := kv.Pair.Validate()
	if err != nil {
		comboError = append(comboError, err.Error())
	}

	_, err = sdk.NewDecFromStr(kv.Multiplier.String())
	if err != nil {
		comboError = append(comboError, err.Error())
	}

	if len(comboError) > 0 {
		return fmt.Errorf("swap_invariant_multiple err: %s", comboError)
	}

	return nil
}

func (kv *EditSwapInvariantsProposal_SwapInvariantMultiple) String() string {
	return fmt.Sprintf(`{ "pair": "%s", "multiplier": "%s" }`, kv.Pair, kv.Multiplier)
}

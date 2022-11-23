package types

import (
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/common"
)

const (
	ProposalTypeCreatePool     = "CreatePool"
	ProposalTypeEditPoolConfig = "EditPoolConfig"
)

var _ govtypes.Content = &CreatePoolProposal{}
var _ govtypes.Content = &EditPoolConfigProposal{}

func init() {
	govtypes.RegisterProposalType(ProposalTypeCreatePool)
	govtypes.RegisterProposalTypeCodec(&CreatePoolProposal{}, "nibiru/CreatePoolProposal")
	govtypes.RegisterProposalType(ProposalTypeEditPoolConfig)
	govtypes.RegisterProposalTypeCodec(&EditPoolConfigProposal{}, "nibiru/EditPoolConfigProposal")
}

// CreatePoolProposal

func (proposal *CreatePoolProposal) ProposalRoute() string {
	return RouterKey
}

func (proposal *CreatePoolProposal) ProposalType() string {
	return ProposalTypeCreatePool
}

func (proposal *CreatePoolProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(proposal); err != nil {
		return err
	}

	assetPair, err := common.NewAssetPair(proposal.Pair)
	if err != nil {
		return err
	}
	pool := &Vpool{
		Pair:              assetPair,
		BaseAssetReserve:  proposal.BaseAssetReserve,
		QuoteAssetReserve: proposal.QuoteAssetReserve,
		Config:            proposal.Config,
	}

	return pool.Validate()
}

// EditPoolConfigProposal

func (proposal *EditPoolConfigProposal) ProposalRoute() string {
	return RouterKey
}

func (proposal *EditPoolConfigProposal) ProposalType() string {
	return ProposalTypeEditPoolConfig
}

func (proposal *EditPoolConfigProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(proposal); err != nil {
		return err
	}

	_, err := common.NewAssetPair(proposal.Pair)
	if err != nil {
		return err
	}

	config := proposal.Config
	return config.Validate()
}

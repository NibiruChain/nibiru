package simulation

import (
	simtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/simulation"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/simulation"
)

const (
	// OpWeightSubmitParamChangeProposal app params key for param change proposal
	OpWeightSubmitParamChangeProposal = "op_weight_submit_param_change_proposal"
	DefaultWeightParamChangeProposal  = 5
)

// ProposalContents defines the module weighted proposals' contents
//
//nolint:staticcheck
func ProposalContents(paramChanges []simtypes.LegacyParamChange) []simtypes.WeightedProposalContent {
	return []simtypes.WeightedProposalContent{
		simulation.NewWeightedProposalContent(
			OpWeightSubmitParamChangeProposal,
			DefaultWeightParamChangeProposal,
			SimulateParamChangeProposalContent(paramChanges),
		),
	}
}

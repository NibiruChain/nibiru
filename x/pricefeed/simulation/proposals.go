package simulation

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// OpWeightSubmitCommunitySpendProposal app params key for community spend proposal
const (
	OpWeightSubmitAddOracleProposal = "op_weight_submit_add_oracle_proposal"
	DefaultWeightAddOracleProposal  = 5
	maxProposalTitleLength          = 20
	maxProposalDescriptionLength    = 10 * maxProposalTitleLength
	maxProposalOracleAccounts       = 10
)

// ProposalContents defines the module weighted proposals' contents
func ProposalContents(k keeper.Keeper) []simtypes.WeightedProposalContent {
	return []simtypes.WeightedProposalContent{
		simulation.NewWeightedProposalContent(
			OpWeightSubmitAddOracleProposal,
			DefaultWeightAddOracleProposal,
			SimulateAddOracleProposalContent(k),
		),
	}
}

// SimulateAddOracleProposalContent generates random add oracle proposal content
func SimulateAddOracleProposalContent(k keeper.Keeper) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		title := simtypes.RandStringOfLength(r, r.Intn(maxProposalTitleLength)+1)
		description := simtypes.RandStringOfLength(r, r.Intn(maxProposalDescriptionLength))
		oracleAddr := simtypes.RandomAccounts(r, r.Intn(maxProposalOracleAccounts)+1)
		var oracles []string
		for _, a := range oracleAddr {
			oracles = append(oracles, a.Address.String())
		}
		var pairs []string
		for len(pairs) < 1 {
			// empty pairs are not allowed, try until we got at least one
			for _, p := range genAssetPairs(r) {
				pairs = append(pairs, p.String())
			}
		}

		return types.NewAddOracleProposal(title, description, oracles, pairs)
	}
}

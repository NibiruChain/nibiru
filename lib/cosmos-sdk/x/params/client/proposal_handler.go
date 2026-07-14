package client

import (
	govclient "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/client/cli"
)

// ProposalHandler is the param change proposal handler.
var ProposalHandler = govclient.NewProposalHandler(cli.NewSubmitParamChangeProposalTxCmd)

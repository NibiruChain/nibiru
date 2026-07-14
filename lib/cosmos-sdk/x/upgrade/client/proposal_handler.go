package client

import (
	govclient "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/client/cli"
)

var (
	LegacyProposalHandler       = govclient.NewProposalHandler(cli.NewCmdSubmitLegacyUpgradeProposal)
	LegacyCancelProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitLegacyCancelUpgradeProposal)
)

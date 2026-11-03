package cli

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/client/cli"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
)

func TestParseArgsToContent(t *testing.T) {
	fs := NewCmdSubmitLegacyUpgradeProposal().Flags()

	proposal := types.SoftwareUpgradeProposal{
		Title:       "proposal title",
		Description: "proposal description",
		Plan: types.Plan{
			Name:   "plan name",
			Height: 123456,
			Info:   "plan info",
		},
	}
	//nolint:errcheck
	fs.Set(cli.FlagTitle, proposal.Title)                                  //nolint:errcheck
	fs.Set(cli.FlagDescription, proposal.Description)                      //nolint:errcheck
	fs.Set(FlagUpgradeHeight, strconv.FormatInt(proposal.Plan.Height, 10)) //nolint:errcheck
	//nolint:errcheck
	fs.Set(FlagUpgradeInfo, proposal.Plan.Info)

	content, err := parseArgsToContent(fs, proposal.Plan.Name)
	require.NoError(t, err)

	p, ok := content.(*types.SoftwareUpgradeProposal)
	require.Equal(t, ok, true)
	require.Equal(t, p.Title, proposal.Title)
	require.Equal(t, p.Description, proposal.Description)
	require.Equal(t, p.Plan.Name, proposal.Plan.Name)
	require.Equal(t, p.Plan.Height, proposal.Plan.Height)
	require.Equal(t, p.Plan.Info, proposal.Plan.Info)
}

func TestParseTimeTargetAndRejectAmbiguousInput(t *testing.T) {
	fs := NewCmdSubmitLegacyUpgradeProposal().Flags()
	require.NoError(t, fs.Set(cli.FlagTitle, "time target"))
	require.NoError(t, fs.Set(cli.FlagDescription, "target selection"))
	require.NoError(t, fs.Set(FlagUpgradeTime, "2027-01-02T03:04:05.123456789Z"))
	content, err := parseArgsToContent(fs, "next")
	require.NoError(t, err)
	plan := content.(*types.SoftwareUpgradeProposal).Plan
	require.Zero(t, plan.Height)
	require.Equal(t, "2027-01-02T03:04:05.123456789Z", plan.Time.Format("2006-01-02T15:04:05.999999999Z07:00"))
	require.NoError(t, fs.Set(FlagUpgradeHeight, "100"))
	_, err = parseArgsToContent(fs, "next")
	require.ErrorContains(t, err, "exactly one")
	require.NoError(t, fs.Set(FlagUpgradeHeight, "0"))
	require.NoError(t, fs.Set(FlagUpgradeTime, "2027-01-02"))
	_, err = parseArgsToContent(fs, "next")
	require.ErrorContains(t, err, "RFC 3339")
}

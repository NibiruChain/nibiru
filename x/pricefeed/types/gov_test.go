package types

import (
	"fmt"
	"io/ioutil"
	"testing"

	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	simappparams "github.com/cosmos/ibc-go/v3/testing/simapp/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestMarshalAddOracleProposal(t *testing.T) {
	t.Log("load example json as bytes")
	_, oracles := sample.PrivKeyAddressPairs(4)
	proposal := AddOracleProposal{
		Title:       "Cataclysm-004",
		Description: "Whitelists Delphi to post prices for OHM and BTC",
		Oracles:     common.AddrsToStrings(oracles...),
		Pairs:       []string{"ohm:usd", "btc:usd"},
	}

	oraclesJSONValue := fmt.Sprintf(`["%s", "%s", "%s", "%s"]`,
		oracles[0], oracles[1], oracles[2], oracles[3])
	proposalJSONString := fmt.Sprintf(`
		{
			"title": "%v",
			"description": "%v",
			"oracles": %v,
			"pairs": ["%v", "%v"]
		}	
		`, proposal.Title, proposal.Description, oraclesJSONValue,
		proposal.Pairs[0], proposal.Pairs[1],
	)
	proposalJSON := sdktestutil.WriteToNewTempFile(
		t, proposalJSONString,
	)
	contents, err := ioutil.ReadFile(proposalJSON.Name())
	assert.NoError(t, err)

	t.Log("Unmarshal json bytes into proposal object; check validity")
	encodingConfig := simappparams.MakeTestEncodingConfig()
	proposal = AddOracleProposal{}
	err = encodingConfig.Marshaler.UnmarshalJSON(contents, &proposal)
	assert.NoError(t, err)
	require.NoError(t, proposal.Validate())
}

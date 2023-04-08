package types

import (
	"fmt"
	"os"
	"testing"

	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simappparams "github.com/cosmos/ibc-go/v4/testing/simapp/params"
	"github.com/gogo/protobuf/jsonpb"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --------------------------------------------------------
// CreatePoolProposal
// --------------------------------------------------------

func TestCreatePoolProposal_ValidateBasic(t *testing.T) {
	type test struct {
		proposal  *CreatePoolProposal
		expectErr bool
	}

	cases := map[string]test{
		"invalid pair": {&CreatePoolProposal{
			Title:       "add proposal",
			Description: "some weird description",
			Pair:        "invalidpair",
		}, true},

		"success": {
			proposal: &CreatePoolProposal{
				Title:             "add proposal",
				Description:       "some weird description",
				Pair:              "valid:pair",
				QuoteAssetReserve: sdk.NewDec(1 * common.TO_MICRO),
				BaseAssetReserve:  sdk.NewDec(1 * common.TO_MICRO),
				Config: MarketConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.10"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.10"),
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
				},
			},
			expectErr: false,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.proposal.ValidateBasic()
			if err == nil && tc.expectErr {
				t.Fatal("error expected")
			} else if err != nil && !tc.expectErr {
				t.Fatal("unexpected error")
			}
		})
	}
}

// --------------------------------------------------------
// EditPoolConfigProposal
// --------------------------------------------------------

func TestEditPoolConfigProposal_ValidateBasic(t *testing.T) {
	type test struct {
		proposal  *EditPoolConfigProposal
		expectErr bool
	}

	validConfig := MarketConfig{
		FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.10"),
		MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
		MaxLeverage:            sdk.MustNewDecFromStr("15"),
		MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.10"),
		TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
	}

	cases := map[string]test{
		"success": {
			proposal: &EditPoolConfigProposal{
				Title:       "edit pool config proposal",
				Description: "proposal description",
				Pair:        "valid:pair",
				Config:      validConfig,
				// MarketConfig.Validate() already has full test coverage
			},
			expectErr: false,
		},

		"invalid pair": {
			proposal: &EditPoolConfigProposal{
				Title:       "edit pool config proposal",
				Description: "proposal description",
				Pair:        "invalidpair",
				Config:      validConfig,
			},
			expectErr: true,
		},

		"err - missing title": {
			proposal: &EditPoolConfigProposal{
				Title:       "",
				Description: "proposal description",
				Pair:        "valid:pair",
				Config:      validConfig,
			},
			expectErr: true,
		},

		"err - missing description": {
			proposal: &EditPoolConfigProposal{
				Title:       "edit pool config proposal",
				Description: "",
				Pair:        "valid:pair",
				Config:      validConfig,
			},
			expectErr: true,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.proposal.ValidateBasic()
			if err == nil && tc.expectErr {
				t.Fatal("error expected")
			} else if err != nil && !tc.expectErr {
				t.Fatal("unexpected error")
			}
		})
	}
}

func TestMarshalProposalEditPoolConfig(t *testing.T) {
	t.Log("load example json as bytes")
	proposal := EditPoolConfigProposal{
		Title:       "Edit market config for NIBI:NUSD",
		Description: "I want to take 100x leverage on my NIBI",
		Pair:        asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
		Config: MarketConfig{
			MaxLeverage:            sdk.MustNewDecFromStr("100"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.10"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.01"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.10"),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
		},
	}
	require.NoError(t, proposal.Config.Validate())

	// proposalJSONString showcases a valid example for the proposal.json file.
	cfg := proposal.Config
	proposalJSONString := fmt.Sprintf(`
	{
		"title": "%v",
		"description": "%v",
		"pair": "%v",
		"config": {
			"max_leverage": "%v",
			"trade_limit_ratio": "%v",
			"fluctuation_limit_ratio": "%v",
			"max_oracle_spread_ratio": "%v",
			"maintenance_margin_ratio": "%v"
		}
	}
	`, proposal.Title, proposal.Description, proposal.Pair,
		cfg.MaxLeverage, cfg.TradeLimitRatio, cfg.FluctuationLimitRatio,
		cfg.MaxOracleSpreadRatio, cfg.MaintenanceMarginRatio,
	)

	tempProposal := EditPoolConfigProposal{}
	err := jsonpb.UnmarshalString(proposalJSONString, &tempProposal)
	require.NoErrorf(t, err, "DEBUG tempProposal: #%v", tempProposal)

	proposalJSON := sdktestutil.WriteToNewTempFile(
		t, proposalJSONString,
	)
	contents, err := os.ReadFile(proposalJSON.Name())
	assert.NoError(t, err)

	t.Log("Unmarshal json bytes into proposal object")
	encodingConfig := simappparams.MakeTestEncodingConfig()

	newProposal := EditPoolConfigProposal{}
	err = encodingConfig.Marshaler.UnmarshalJSON(contents, &newProposal)
	assert.NoErrorf(t, err, "DEBUG proposalJSONString: #%v", proposalJSONString)
	require.NoError(t, newProposal.ValidateBasic(), newProposal.String())
}

// --------------------------------------------------------
// EditSwapInvariantsProposal
// --------------------------------------------------------

func TestMarshalProposalEditSwapInvariants(t *testing.T) {
	t.Log("load example json as bytes")
	proposal := EditSwapInvariantsProposal{
		Title:       "NIP-4: Change the swap invariant for ATOM, OSMO, and BTC.",
		Description: "increase swap invariant for many virtual pools",
		SwapInvariantMaps: []EditSwapInvariantsProposal_SwapInvariantMultiple{
			{Pair: "uatom:unusd", Multiplier: sdk.NewDec(2)},
			{Pair: "uosmo:unusd", Multiplier: sdk.NewDec(3)},
			{Pair: "ubtc:unusd", Multiplier: sdk.NewDec(99)},
		},
	}
	require.NoError(t, proposal.ValidateBasic())

	// proposalJSONString showcases a valid example for the proposal.json file.
	maps := proposal.SwapInvariantMaps
	proposalJSONString := fmt.Sprintf(`
	{
	  "title": "%v",
	  "description": "%v",
	  "swap_invariant_maps": [
		%s,
		%s,
		%s
	  ]
	}
	`, proposal.Title, proposal.Description, maps[0].String(), maps[1].String(), maps[2].String(),
	)

	tempProposal := EditSwapInvariantsProposal{}
	err := jsonpb.UnmarshalString(proposalJSONString, &tempProposal)
	require.NoErrorf(t, err, "DEBUG tempProposal: %v \nproposalJSONString: %v",
		tempProposal, proposalJSONString)

	proposalJSON := sdktestutil.WriteToNewTempFile(
		t, proposalJSONString,
	)
	contents, err := os.ReadFile(proposalJSON.Name())
	assert.NoError(t, err)

	t.Log("Unmarshal json bytes into proposal object")
	encodingConfig := simappparams.MakeTestEncodingConfig()

	newProposal := EditSwapInvariantsProposal{}
	err = encodingConfig.Marshaler.UnmarshalJSON(contents, &newProposal)
	assert.NoErrorf(t, err, "DEBUG proposalJSONString: #%v", proposalJSONString)
	require.NoError(t, newProposal.ValidateBasic(), newProposal.String())
}

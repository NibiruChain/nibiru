package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

func TestMsgServer_DeprecatedTxHandlers(t *testing.T) {
	input, msgServer := Setup(t)

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "aggregate prevote deprecated",
			call: func() error {
				_, err := msgServer.AggregateExchangeRatePrevote(
					sdk.WrapSDKContext(input.Ctx),
					&types.MsgAggregateExchangeRatePrevote{
						Hash:      "not-even-hex",
						Feeder:    "invalid-feeder",
						Validator: "invalid-validator",
					},
				)
				return err
			},
		},
		{
			name: "aggregate vote deprecated",
			call: func() error {
				_, err := msgServer.AggregateExchangeRateVote(
					sdk.WrapSDKContext(input.Ctx),
					&types.MsgAggregateExchangeRateVote{
						Salt:          "irrelevant",
						ExchangeRates: "not-a-valid-tuple",
						Feeder:        "invalid-feeder",
						Validator:     "invalid-validator",
					},
				)
				return err
			},
		},
		{
			name: "delegate feed consent deprecated",
			call: func() error {
				_, err := msgServer.DelegateFeedConsent(
					sdk.WrapSDKContext(input.Ctx),
					&types.MsgDelegateFeedConsent{
						Operator: "invalid-operator",
						Delegate: "invalid-delegate",
					},
				)
				return err
			},
		},
		{
			name: "edit oracle params deprecated",
			call: func() error {
				_, err := msgServer.EditOracleParams(
					sdk.WrapSDKContext(input.Ctx),
					&types.MsgEditOracleParams{
						Sender: "invalid-sender",
						Params: &types.OracleParamsMsg{VotePeriod: 100},
					},
				)
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.ErrorIs(t, err, types.ErrOracleDeprecated)
		})
	}
}

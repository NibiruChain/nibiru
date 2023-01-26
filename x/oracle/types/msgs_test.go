package types_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/oracle/types"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMsgFeederDelegation(t *testing.T) {
	addrs := []sdk.AccAddress{
		sdk.AccAddress([]byte("addr1_______________")),
		sdk.AccAddress([]byte("addr2_______________")),
	}

	tests := []struct {
		delegator  sdk.ValAddress
		delegate   sdk.AccAddress
		expectPass bool
	}{
		{sdk.ValAddress(addrs[0]), addrs[1], true},
		{sdk.ValAddress{}, addrs[1], false},
		{sdk.ValAddress(addrs[0]), sdk.AccAddress{}, false},
		{nil, nil, false},
	}

	for i, tc := range tests {
		msg := types.NewMsgDelegateFeedConsent(tc.delegator, tc.delegate)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestMsgAggregateExchangeRatePrevote(t *testing.T) {
	addrs := []sdk.AccAddress{
		sdk.AccAddress([]byte("addr1_______________")),
	}

	exchangeRates := sdk.DecCoins{sdk.NewDecCoinFromDec(denoms.USDC, sdk.OneDec()), sdk.NewDecCoinFromDec(denoms.NUSD, sdk.NewDecWithPrec(32121, 1))}
	bz := types.GetAggregateVoteHash("1", exchangeRates.String(), sdk.ValAddress(addrs[0]))

	tests := []struct {
		hash          types.AggregateVoteHash
		exchangeRates sdk.DecCoins
		voter         sdk.AccAddress
		expectPass    bool
	}{
		{bz, exchangeRates, addrs[0], true},
		{bz[1:], exchangeRates, addrs[0], false},
		{bz, exchangeRates, sdk.AccAddress{}, false},
		{types.AggregateVoteHash{}, exchangeRates, addrs[0], false},
	}

	for i, tc := range tests {
		msg := types.NewMsgAggregateExchangeRatePrevote(tc.hash, tc.voter, sdk.ValAddress(tc.voter))
		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestMsgAggregateExchangeRateVote(t *testing.T) {
	addrs := []sdk.AccAddress{
		sdk.AccAddress("addr1_______________"),
	}

	exchangeRates := types.ExchangeRateTuples{
		{
			Pair:         "FOO:USD",
			ExchangeRate: sdk.MustNewDecFromStr("1.0"),
		},
		{
			Pair:         "BAR:USD",
			ExchangeRate: sdk.MustNewDecFromStr("1232.132"),
		},
	}

	abstainExchangeRates := types.ExchangeRateTuples{
		{
			Pair:         "FOO:USD",
			ExchangeRate: sdk.ZeroDec(),
		},
		{
			Pair:         "BAR:USD",
			ExchangeRate: sdk.MustNewDecFromStr("1232.132"),
		},
	}

	tests := []struct {
		voter         sdk.AccAddress
		validator     sdk.ValAddress
		salt          string
		exchangeRates types.ExchangeRateTuples
		expectPass    bool
	}{
		{addrs[0], sdk.ValAddress(addrs[0]), "123", exchangeRates, true},
		{addrs[0], sdk.ValAddress(addrs[0]), "123", abstainExchangeRates, true},
		{sdk.AccAddress{}, sdk.ValAddress(addrs[0]), "123", exchangeRates, false},
		{addrs[0], sdk.ValAddress(addrs[0]), "123", types.ExchangeRateTuples{}, false},
		{addrs[0], sdk.ValAddress{}, "123", abstainExchangeRates, false},
		{addrs[0], sdk.ValAddress(addrs[0]), "", abstainExchangeRates, false},
	}

	for i, tc := range tests {
		exchangeRates, err := tc.exchangeRates.ToString()
		require.NoError(t, err)
		msg := types.NewMsgAggregateExchangeRateVote(tc.salt, exchangeRates, tc.voter, tc.validator)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

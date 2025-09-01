package keeper_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
)

var (
	invalidAddress = "0xinvalidHexAddress"

	validFeeToken = types.FeeToken{
		Name:         "Valid Fee Token",
		Erc20Address: evmtest.NewEthPrivAcc().EthAddr.String(),
	}

	anotherValidFeeToken = types.FeeToken{
		Name:         "Another Valid Fee Token",
		Erc20Address: evmtest.NewEthPrivAcc().EthAddr.String(),
		// PoolAddress:  evmtest.NewEthPrivAcc().EthAddr.String(),
	}

	invalidFeeToken = types.FeeToken{
		Name:         "Invalid Fee Token",
		Erc20Address: invalidAddress,
	}

	validFeeTokens = []types.FeeToken{
		validFeeToken, anotherValidFeeToken,
	}

	invalidFeeTokens = []types.FeeToken{
		validFeeToken, invalidFeeToken,
	}
)

func TestSetFeeToken(t *testing.T) {
	testCases := []struct {
		name        string
		feeToken    types.FeeToken
		expectedErr error
	}{
		{
			name:        "valid fee token",
			feeToken:    validFeeToken,
			expectedErr: nil,
		},
		{
			name:        "invalid fee token",
			feeToken:    invalidFeeToken,
			expectedErr: fmt.Errorf("invalid fee token address %s: must be a valid hex address", invalidFeeToken.Erc20Address),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

			if tc.expectedErr != nil {
				err := nibiruApp.GasTokenKeeper.SetFeeToken(ctx, tc.feeToken)
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErr.Error())
				return
			}

			err := nibiruApp.GasTokenKeeper.SetFeeToken(
				ctx, tc.feeToken)
			require.NoError(t, err)

			feeToken, err := nibiruApp.GasTokenKeeper.GetFeeToken(ctx, tc.feeToken.Erc20Address)
			require.NoError(t, err)
			require.Equal(t, feeToken, tc.feeToken)

		})
	}
}

func TestSetFeeTokens(t *testing.T) {
	testCases := []struct {
		name        string
		feeTokens   []types.FeeToken
		expectedErr error
	}{
		{
			name:        "valid fee tokens",
			feeTokens:   validFeeTokens,
			expectedErr: nil,
		},
		{
			name:        "invalid fee tokens",
			feeTokens:   invalidFeeTokens,
			expectedErr: fmt.Errorf("invalid fee token address %s: must be a valid hex address", invalidFeeTokens[1].Erc20Address),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

			if tc.expectedErr != nil {
				err := nibiruApp.GasTokenKeeper.SetFeeTokens(ctx, tc.feeTokens)
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErr.Error())
				return
			}

			err := nibiruApp.GasTokenKeeper.SetFeeTokens(
				ctx, tc.feeTokens)
			require.NoError(t, err)

			feeTokens := nibiruApp.GasTokenKeeper.GetFeeTokens(ctx)
			require.NoError(t, err)
			sortFeeTokens(feeTokens)
			sortFeeTokens(tc.feeTokens)
			require.Equal(t, feeTokens, tc.feeTokens)

		})
	}
}

func TestRemoveFeeToken(t *testing.T) {
	testCases := []struct {
		name        string
		expectedErr error
	}{
		{
			name:        "successfully remove fee token",
			expectedErr: nil,
		},
		{
			name:        "fail to invalid fee token address",
			expectedErr: fmt.Errorf("invalid fee token address %s: must be a valid hex address", "0xinvalidHexAddress"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

			err := nibiruApp.GasTokenKeeper.SetFeeTokens(
				ctx, validFeeTokens)

			feeTokens := nibiruApp.GasTokenKeeper.GetFeeTokens(ctx)
			require.NoError(t, err)
			sortFeeTokens(feeTokens)
			sortFeeTokens(validFeeTokens)
			require.Equal(t, feeTokens, validFeeTokens)

			if tc.expectedErr != nil {
				err := nibiruApp.GasTokenKeeper.RemoveFeeToken(ctx, invalidAddress)
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErr.Error())
				return
			}

			err = nibiruApp.GasTokenKeeper.RemoveFeeToken(ctx, validFeeTokens[0].Erc20Address)
			require.NoError(t, err)
		})
	}
}

func sortFeeTokens(tokens []types.FeeToken) {
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].Erc20Address < tokens[j].Erc20Address
	})
}

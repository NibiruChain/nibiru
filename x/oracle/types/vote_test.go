package types_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"

	"github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/stretchr/testify/require"
)

func TestParseExchangeRateTuples(t *testing.T) {
	t.Run("inverse", func(t *testing.T) {
		tuples := types.ExchangeRateTuples{
			{
				Pair:         "BTC:USD",
				ExchangeRate: sdk.MustNewDecFromStr("40000.00"),
			},

			{
				Pair:         "ETH:USD",
				ExchangeRate: sdk.MustNewDecFromStr("4000.00"),
			},
		}

		tuplesStr, err := tuples.ToString()
		require.NoError(t, err)

		parsedTuples := new(types.ExchangeRateTuples)
		require.NoError(t, parsedTuples.FromString(tuplesStr))

		require.Equal(t, tuples, *parsedTuples)
	})

	t.Run("check duplicates", func(t *testing.T) {

	})
}

func TestExchangeRateTuple(t *testing.T) {
	t.Run("inverse", func(t *testing.T) {
		exchangeRate := types.ExchangeRateTuple{
			Pair:         "BTC:USD",
			ExchangeRate: sdk.MustNewDecFromStr("40000.00"),
		}
		exchangeRateStr, err := exchangeRate.ToString()
		require.NoError(t, err)

		parsedExchangeRate := new(types.ExchangeRateTuple)
		require.NoError(t, parsedExchangeRate.FromString(exchangeRateStr))

		require.Equal(t, exchangeRate, *parsedExchangeRate)
	})

	t.Run("invalid size", func(t *testing.T) {
		// TODO(mercilex)
	})

	t.Run("invalid delimiters", func(t *testing.T) {
		// TODO(mercilex)
	})

	t.Run("invalid format", func(t *testing.T) {
		// TODO(mercilex)
	})

}

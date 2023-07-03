package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestExchangeRateTuples_ToString(t *testing.T) {
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

		parsedTuples, err := types.NewExchangeRateTuplesFromString(tuplesStr)
		require.NoError(t, err)

		require.Equal(t, tuples, parsedTuples)
	})

	t.Run("check duplicates", func(t *testing.T) {
		tuples := types.ExchangeRateTuples{
			{
				Pair:         "BTC:USD",
				ExchangeRate: sdk.MustNewDecFromStr("40000.00"),
			},

			{
				Pair:         "BTC:USD",
				ExchangeRate: sdk.MustNewDecFromStr("4000.00"),
			},
		}

		tuplesStr, err := tuples.ToString()
		require.NoError(t, err)

		_, err = types.NewExchangeRateTuplesFromString(tuplesStr)
		require.ErrorContains(t, err, "found duplicate")
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

		parsedExchangeRate, err := types.NewExchangeRateTupleFromString(exchangeRateStr)
		require.NoError(t, err)

		require.Equal(t, exchangeRate, parsedExchangeRate)
	})

	t.Run("invalid size", func(t *testing.T) {
		_, err := types.NewExchangeRateTupleFromString("00")
		require.ErrorContains(t, err, "invalid string length")
	})

	t.Run("invalid delimiters", func(t *testing.T) {
		_, err := types.NewExchangeRateTupleFromString("|1000.0,nibi:usd|")
		require.ErrorContains(t, err, "invalid ExchangeRateTuple delimiters")
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := types.NewExchangeRateTupleFromString("(1000.0,nibi:usd,1000.0)")
		require.ErrorContains(t, err, "invalid ExchangeRateTuple format")
	})
}

package localnet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAccountSequenceMismatch(t *testing.T) {
	rawLog := "account sequence mismatch, expected 27, got 26: incorrect account sequence"

	expected, got, ok := parseAccountSequenceMismatch(rawLog)

	require.True(t, ok)
	require.Equal(t, "27", expected)
	require.Equal(t, "26", got)
}

func TestParseAccountSequenceMismatchRejectsOtherErrors(t *testing.T) {
	expected, got, ok := parseAccountSequenceMismatch("out of gas")

	require.False(t, ok)
	require.Empty(t, expected)
	require.Empty(t, got)
}

func TestSetTxRetryFlags(t *testing.T) {
	require.Equal(t,
		[]string{"send", "--offline=true", "--account-number=7", "--sequence=27"},
		setTxRetryFlags([]string{"send"}, "7", "27"),
	)
	require.Equal(t,
		[]string{"send", "--account-number=7", "--sequence=28", "--offline=true"},
		setTxRetryFlags([]string{"send", "--account-number=6", "--sequence=27"}, "7", "28"),
	)
	require.Equal(t,
		[]string{"send", "--account-number", "8", "--sequence", "29", "--offline=true"},
		setTxRetryFlags([]string{"send", "--account-number", "6", "--sequence", "27"}, "8", "29"),
	)
}

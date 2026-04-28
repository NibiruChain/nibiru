package nutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAccountSequenceMismatch(t *testing.T) {
	rawLog := "account sequence mismatch, expected 27, got 26: incorrect account sequence"

	expected, got, ok := ParseAccountSequenceMismatch(rawLog)

	require.True(t, ok)
	require.Equal(t, uint64(27), expected)
	require.Equal(t, uint64(26), got)
}

func TestParseAccountSequenceMismatchRejectsOtherErrors(t *testing.T) {
	expected, got, ok := ParseAccountSequenceMismatch("out of gas")

	require.False(t, ok)
	require.Zero(t, expected)
	require.Zero(t, got)
}

package localnet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

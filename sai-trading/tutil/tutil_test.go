package tutil_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/sai-trading/tutil"
	"github.com/stretchr/testify/require"
)

func TestA_EnsureLocalBlockchainRunning(t *testing.T) {
	err := tutil.EnsureLocalBlockchain()
	require.NoError(t, err)
}

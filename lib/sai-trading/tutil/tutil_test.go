package tutil_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/lib/sai-trading/tutil"
)

func TestA_EnsureLocalBlockchainRunning(t *testing.T) {
	if err := tutil.EnsureLocalBlockchain(); err != nil {
		t.Skipf("localnet not running: %v", err)
	}
}

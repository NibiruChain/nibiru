package evmtest_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func TestLoadContracts(t *testing.T) {
	evmtest.SmartContract_FunToken.Load(t)
}

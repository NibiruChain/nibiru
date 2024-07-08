package embeds_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/x/evm/embeds"
)

func TestLoadContracts(t *testing.T) {
	for _, tc := range []embeds.SmartContractFixture{
		embeds.SmartContract_TestERC20,
		embeds.SmartContract_ERC20Minter,
		embeds.SmartContract_FunToken,
	} {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := tc.Load()
			assert.NoError(t, err)
		})
	}
}
